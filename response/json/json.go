package json

import (
	"bytes"
	"encoding/json"

	"github.com/valyala/bytebufferpool"
)

const (
	// ContentType the key for the engine, the user/dev can still use its own
	ContentType = "application/json"
)

// Engine the response engine which renders a JSON 'object'
type Engine struct {
	config     Config
	bufferPool bytebufferpool.Pool
}

// New returns a new json response engine
func New(cfg ...Config) *Engine {
	c := DefaultConfig().Merge(cfg)
	return &Engine{config: c}
}

// Response accepts the 'object' value and converts it to bytes in order to be 'renderable'
// implements the Q.ResponseEngine
func (e *Engine) Response(val interface{}, options ...map[string]interface{}) ([]byte, error) {
	w := e.bufferPool.Get()
	defer e.bufferPool.Put(w)
	if e.config.StreamingJSON {

		if len(e.config.Prefix) > 0 {
			w.Write(e.config.Prefix)
		}
		err := json.NewEncoder(w).Encode(val)
		return w.Bytes(), err
	}

	var result []byte
	var err error

	if e.config.Indent {
		result, err = json.MarshalIndent(val, "", "  ")
		result = append(result, '\n')
	} else {
		result, err = json.Marshal(val)
	}
	if err != nil {
		return nil, err
	}

	if e.config.UnEscapeHTML {
		result = bytes.Replace(result, []byte("\\u003c"), []byte("<"), -1)
		result = bytes.Replace(result, []byte("\\u003e"), []byte(">"), -1)
		result = bytes.Replace(result, []byte("\\u0026"), []byte("&"), -1)
	}

	if len(e.config.Prefix) > 0 {
		w.Write(e.config.Prefix)
	}

	w.Write(result)
	return w.Bytes(), nil
}
