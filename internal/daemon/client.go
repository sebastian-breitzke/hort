package daemon

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/s16e/hort/internal/store"
)

// Client is a short-lived Unix-socket client for the Hort daemon.
type Client struct {
	conn    net.Conn
	reader  *bufio.Reader
	writer  *bufio.Writer
	timeout time.Duration
}

// Dial tries to connect to the daemon. Returns a *Client ready to use, or an
// error if the socket is absent / unreachable.
func Dial() (*Client, error) {
	sockPath, err := SocketPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(sockPath); err != nil {
		return nil, err
	}
	conn, err := net.DialTimeout("unix", sockPath, 250*time.Millisecond)
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:    conn,
		reader:  bufio.NewReader(conn),
		writer:  bufio.NewWriter(conn),
		timeout: 5 * time.Second,
	}, nil
}

// Available is a fast check — does a daemon socket exist and accept a connection?
func Available() bool {
	c, err := Dial()
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// Close terminates the connection.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// Call sends a Request and returns the Response body.
func (c *Client) Call(req Request) (*Response, error) {
	_ = c.conn.SetDeadline(time.Now().Add(c.timeout))
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if _, err := c.writer.Write(data); err != nil {
		return nil, err
	}
	if err := c.writer.WriteByte('\n'); err != nil {
		return nil, err
	}
	if err := c.writer.Flush(); err != nil {
		return nil, err
	}
	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	if !resp.OK {
		return &resp, errors.New(resp.Error)
	}
	return &resp, nil
}

// DecodeEntries extracts a `[]store.EntryInfo` from a list/describe response.
func DecodeEntries(resp *Response) ([]store.EntryInfo, error) {
	raw, ok := resp.Result["entries"]
	if !ok {
		return nil, errors.New("missing 'entries' in response")
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var entries []store.EntryInfo
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}
