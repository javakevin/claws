package main

import (
    "time"

	"net/http"

	"github.com/gorilla/websocket"
)

// WebSocket is a wrapper around a gorilla.WebSocket for claws.
type WebSocket struct {
	conn      *websocket.Conn
	writeChan chan string
	closed    bool
	url       string
}

// URL returns the URL of the WebSocket.
func (w *WebSocket) URL() string {
	return w.url
}

// ReadChannel retrieves a channel from which to read messages out of.
func (w *WebSocket) ReadChannel() <-chan string {
	ch := make(chan string, 16)
	go w.readChannel(ch)
	return ch
}

// Write writes a message to the WebSocket
func (w *WebSocket) Write(msg string) {
	w.writeChan <- msg
}

func (w *WebSocket) readChannel(c chan<- string) {
	for {
		_type, msg, err := w.conn.ReadMessage()
		if err != nil {
			if !w.closed {
				state.Error(err.Error())
				w.close(c)
			}
			return
		}

		switch _type {
		case websocket.TextMessage, websocket.BinaryMessage:
			c <- string(msg)
		case websocket.CloseMessage:
			cl := "Closed WebSocket"
			if len(msg) > 0 {
				cl += " " + string(msg)
			}
			state.Debug(cl)
			w.close(c)
			return
		case websocket.PingMessage, websocket.PongMessage:
			if len(msg) > 0 {
				state.Debug("Ping/pong with " + string(msg))
			}
		}
	}
}

func (w *WebSocket) writePump() {
	for msg := range w.writeChan {
		err := w.conn.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			state.Error(err.Error())
			w.Close()
			return
		}
	}
}

// Close closes the WebSocket connection.
func (w *WebSocket) Close() error {
	if w == nil {
		return nil
	}
	if w.closed {
		return nil
	}
	w.closed = true
	if state.Conn == w {
		state.Conn = nil
	}
	close(w.writeChan)
	return w.conn.Close()
}

// close finalises the WebSocket connection.
func (w *WebSocket) close(c chan<- string) error {
	close(c)
	return w.Close()
}

func (w *WebSocket) doPing() {
    var data = []byte("hello")
    if err := w.conn.WriteMessage(websocket.PingMessage, data); err != nil {
        state.Error(err.Error())
    }
}

func (w *WebSocket) pingTimer() {
    timer := time.NewTicker(30 * time.Second);
    for {
        select {
        case <-timer.C:
            w.doPing();
        }
    }
}

// WebSocketResponseError is the error returned when there is an error in
// CreateWebSocket.
type WebSocketResponseError struct {
	Err  error
	Resp *http.Response
}

func (w WebSocketResponseError) Error() string {
	return w.Err.Error()
}

// CreateWebSocket initialises a new WebSocket connection.
func CreateWebSocket(url string) (*WebSocket, error) {
	state.Debug("Starting WebSocket connection to " + url)

    var header = map[string][]string{
        "Content-Type": {"application/json"},
        "Accept": {"application/json"},
    }
	conn, resp, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		return nil, WebSocketResponseError{
			Err:  err,
			Resp: resp,
		}
	}

	ws := &WebSocket{
		conn:      conn,
		writeChan: make(chan string, 128),
		url:       url,
	}

	go ws.writePump()

    go ws.pingTimer()

	return ws, nil
}
