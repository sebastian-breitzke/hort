package daemon

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"github.com/s16e/hort/internal/helptext"
	"github.com/s16e/hort/internal/store"
	"github.com/s16e/hort/internal/vault"
)

// Server is the Hort background daemon. It holds live session state in memory
// and answers multi-source read/write requests over a Unix socket.
type Server struct {
	socketPath string
	listener   net.Listener
	mu         sync.Mutex
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

// Serve accepts connections until the listener is closed.
func (s *Server) Serve() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		go s.handle(conn)
	}
}

// Close stops the listener and removes the socket file.
func (s *Server) Close() error {
	if s.listener == nil {
		return nil
	}
	err := s.listener.Close()
	_ = os.Remove(s.socketPath)
	return err
}

func (s *Server) handle(conn net.Conn) {
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
		resp := s.dispatch(&req)
		if err := writeResponse(writer, resp); err != nil {
			return
		}
	}
}

func (s *Server) dispatch(req *Request) Response {
	s.mu.Lock()
	defer s.mu.Unlock()

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
