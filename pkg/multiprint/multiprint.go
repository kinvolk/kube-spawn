package multiprint

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"
)

type message struct {
	prefix string
	value  []byte
}

type Multiprint struct {
	ctx         context.Context
	messageChan chan message
}

type Writer struct {
	ctx         context.Context
	messageChan chan message
	prefix      string
	cancelled   bool
}

func New(ctx context.Context) *Multiprint {
	return &Multiprint{
		ctx:         ctx,
		messageChan: make(chan message),
	}
}

func (m *Multiprint) RunPrintLoop() {
	go func() {
		var previousPrefix, prefix string
		for {
			select {
			case <-m.ctx.Done():
				return
			case message, ok := <-m.messageChan:
				if !ok {
					return
				}
				if previousPrefix != message.prefix {
					previousPrefix = message.prefix
					prefix = message.prefix
				}
				scanner := bufio.NewScanner(bytes.NewBuffer(message.value))
				for scanner.Scan() {
					text := strings.TrimSpace(scanner.Text())
					if text == "" {
						continue
					}
					fmt.Printf("%s%s\n", prefix, text)
					prefix = strings.Repeat(" ", len(prefix))
				}
			}
		}
	}()
}

func (m *Multiprint) NewWriter(prefix string) *Writer {
	writer := &Writer{
		ctx:         m.ctx,
		messageChan: m.messageChan,
		prefix:      prefix,
	}
	go func() {
		select {
		case <-writer.ctx.Done():
			writer.cancelled = true
		}
	}()
	return writer
}

func (w *Writer) Write(p []byte) (n int, err error) {
	if w.cancelled {
		return 0, fmt.Errorf("writer was cancelled")
	}
	w.messageChan <- message{prefix: w.prefix, value: p}
	return len(p), nil
}
