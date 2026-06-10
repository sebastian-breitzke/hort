package daemon

import (
	"bufio"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/s16e/hort/internal/helptext"
	"github.com/s16e/hort/internal/store"
	"github.com/s16e/hort/internal/vault"
)

// Server is the Hort background daemon. It holds live session state in memory
// and answers multi-source read/write requests over a Unix socket. An optional
// TCP listener serves a restricted, token-authenticated read-only subset for
// clients that cannot reach the Unix socket (e.g. Docker containers on macOS,
// where Docker Desktop does not pass host Unix sockets through its VM).
type Server struct {
	socketPath  string
	listener    net.Listener
	tcpAddr     string
	tcpToken    string
	tcpListener net.Listener
}

// tcpMethods is the allowlist for the restricted TCP transport: reads only.
// Writes, deletes and discovery (list/describe leak entry names) stay
// Unix-socket-only.
var tcpMethods = map[string]bool{
	MethodGetSecret: true,
	MethodGetConfig: true,
	MethodStatus:    true,
}

// NewServer constructs a Server. Caller must call Start and Close.
func NewServer(socketPath string) *Server {
	return &Server{socketPath: socketPath}
}

// Start listens on the configured socket. Any existing socket file at the same
// path is removed first (stale from a prior crash).
func (s *Server) Start() error {
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cleaning stale socket: %w", err)
	}
	l, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.socketPath, err)
	}
	if err := os.Chmod(s.socketPath, 0600); err != nil {
		_ = l.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}
	s.listener = l
	return nil
}

// StartTCP additionally listens on a TCP address. Connections on this
// listener must authenticate every request with the given token and are
// limited to the read-only tcpMethods allowlist. Call after Start, before
// Serve.
func (s *Server) StartTCP(addr, token string) error {
	if token == "" {
		return errors.New("tcp listener requires a non-empty token")
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on tcp %s: %w", addr, err)
	}
	s.tcpAddr = addr
	s.tcpToken = token
	s.tcpListener = l
	return nil
}

// Serve accepts connections until the listeners are closed.
func (s *Server) Serve() error {
	if s.tcpListener != nil {
		go s.serveListener(s.tcpListener, true)
	}
	return s.serveListener(s.listener, false)
}

func (s *Server) serveListener(l net.Listener, restricted bool) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		go s.handle(conn, restricted)
	}
}

// Close stops the listeners and removes the socket file.
func (s *Server) Close() error {
	if s.tcpListener != nil {
		_ = s.tcpListener.Close()
	}
	if s.listener == nil {
		return nil
	}
	err := s.listener.Close()
	_ = os.Remove(s.socketPath)
	return err
}

func (s *Server) handle(conn net.Conn, restricted bool) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				writeError(writer, err.Error())
			}
			return
		}
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			writeError(writer, fmt.Sprintf("invalid request: %v", err))
			continue
		}
		if restricted {
			if denied, resp := s.gateRestricted(&req); denied {
				if err := writeResponse(writer, resp); err != nil {
					return
				}
				continue
			}
		}
		resp := s.dispatch(&req)
		if err := writeResponse(writer, resp); err != nil {
			return
		}
	}
}

// gateRestricted enforces token auth and the read-only method allowlist for
// TCP connections. Token check first so unauthenticated probes learn nothing
// about which methods exist.
func (s *Server) gateRestricted(req *Request) (bool, Response) {
	if subtle.ConstantTimeCompare([]byte(req.Token), []byte(s.tcpToken)) != 1 {
		return true, errResp(errors.New("invalid or missing token"))
	}
	if !tcpMethods[req.Method] {
		return true, errResp(fmt.Errorf("method %q not allowed over tcp", req.Method))
	}
	return false, Response{}
}

// dispatch handles requests concurrently without daemon-level locking: writes
// are serialized per vault by the flock in vault.UpdateVault, and readers are
// safe against concurrent writers because vault files are replaced atomically
// (temp file + rename). This keeps a slow vault open — e.g. a mounted source
// on a sleeping external volume — from stalling requests that don't touch it.
func (s *Server) dispatch(req *Request) Response {
	st, err := store.NewFromSession()
	if err != nil {
		return errResp(err)
	}

	switch req.Method {
	case MethodStatus:
		primaryRef, _ := vault.PrimaryRef()
		items := []map[string]any{{
			"name":     primaryRef.Name,
			"path":     primaryRef.Path,
			"kdf":      "argon2id",
			"unlocked": vault.IsUnlockedFor(primaryRef),
			"primary":  true,
		}}
		records, _ := vault.ListSources()
		for _, rec := range records {
			ref, err := vault.RefFromRecord(rec)
			if err != nil {
				continue
			}
			kdf := "argon2id"
			if rec.KDF == vault.KDFRawKey {
				kdf = "raw"
			}
			items = append(items, map[string]any{
				"name":     rec.Name,
				"path":     rec.Path,
				"kdf":      kdf,
				"unlocked": vault.IsUnlockedFor(ref),
				"primary":  false,
			})
		}
		return okResp(map[string]any{"sources": items})

	case MethodGetSecret, MethodGetConfig:
		name := strParam(req, "name")
		env := strParam(req, "env")
		context := strParam(req, "context")
		source := strParam(req, "source")
		isSecret := req.Method == MethodGetSecret

		if source != "" {
			entryType := "secret"
			if !isSecret {
				entryType = "config"
			}
			val, err := st.GetFrom(source, name, env, context, entryType)
			if err != nil {
				return errResp(err)
			}
			return okResp(map[string]any{"value": val, "source": source})
		}
		var val, src string
		if isSecret {
			val, src, err = st.GetSecret(name, env, context)
		} else {
			val, src, err = st.GetConfig(name, env, context)
		}
		if err != nil {
			return errResp(err)
		}
		return okResp(map[string]any{"value": val, "source": src})

	case MethodSetSecret, MethodSetConfig:
		name := strParam(req, "name")
		value := strParam(req, "value")
		env := strParam(req, "env")
		context := strParam(req, "context")
		desc := strParam(req, "description")
		source := strParam(req, "source")
		if req.Method == MethodSetSecret {
			err = st.SetSecret(source, name, value, env, context, desc)
		} else {
			err = st.SetConfig(source, name, value, env, context, desc)
		}
		if err != nil {
			return errResp(err)
		}
		return okResp(nil)

	case MethodDelete:
		name := strParam(req, "name")
		env := strParam(req, "env")
		context := strParam(req, "context")
		source := strParam(req, "source")
		if err := st.Delete(source, name, env, context); err != nil {
			return errResp(err)
		}
		return okResp(nil)

	case MethodList:
		typeFilter := strParam(req, "type")
		entries, err := st.List(typeFilter)
		if err != nil {
			return errResp(err)
		}
		return okResp(map[string]any{"entries": entries})

	case MethodDescribe:
		name := strParam(req, "name")
		entries, err := st.Describe(name)
		if err != nil {
			return errResp(err)
		}
		return okResp(map[string]any{"entries": entries})

	case MethodHelp:
		return okResp(map[string]any{"text": helptext.HelpText})

	default:
		return errResp(fmt.Errorf("unknown method %q", req.Method))
	}
}

func okResp(result map[string]any) Response {
	return Response{OK: true, Result: result}
}

func errResp(err error) Response {
	return Response{OK: false, Error: err.Error()}
}

func strParam(req *Request, key string) string {
	if req.Params == nil {
		return ""
	}
	v, ok := req.Params[key]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func writeResponse(w *bufio.Writer, resp Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	if err := w.WriteByte('\n'); err != nil {
		return err
	}
	return w.Flush()
}

func writeError(w *bufio.Writer, msg string) {
	resp := Response{OK: false, Error: msg}
	_ = writeResponse(w, resp)
}
