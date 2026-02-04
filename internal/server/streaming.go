package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type StreamWriter interface {
	Write(data interface{}) error
	Flush() error
	Close() error
}

type JSONStreamWriter struct {
	writer  io.Writer
	flusher http.Flusher
	encoder *json.Encoder
	mu      sync.Mutex
	closed  bool
	count   int
}

func NewJSONStreamWriter(w http.ResponseWriter) *JSONStreamWriter {
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Transfer-Encoding", "chunked")
	flusher, _ := w.(http.Flusher)
	return &JSONStreamWriter{writer: w, flusher: flusher, encoder: json.NewEncoder(w)}
}

func (s *JSONStreamWriter) Write(data interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("stream closed")
	}
	if err := s.encoder.Encode(data); err != nil {
		return err
	}
	s.count++
	if s.flusher != nil {
		s.flusher.Flush()
	}
	return nil
}

func (s *JSONStreamWriter) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.flusher != nil {
		s.flusher.Flush()
	}
	return nil
}

func (s *JSONStreamWriter) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *JSONStreamWriter) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count
}

type StreamReader struct {
	reader  io.Reader
	scanner *bufio.Scanner
	decoder *json.Decoder
}

func NewStreamReader(r io.Reader) *StreamReader {
	return &StreamReader{reader: r, scanner: bufio.NewScanner(r), decoder: json.NewDecoder(r)}
}

func (s *StreamReader) ReadLine() ([]byte, error) {
	if s.scanner.Scan() {
		return s.scanner.Bytes(), nil
	}
	if err := s.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (s *StreamReader) ReadJSON(v interface{}) error {
	return s.decoder.Decode(v)
}

type StreamProcessor struct {
	batchSize int
	timeout   int
}

func NewStreamProcessor(batchSize, timeout int) *StreamProcessor {
	if batchSize <= 0 {
		batchSize = 100
	}
	if timeout <= 0 {
		timeout = 30
	}
	return &StreamProcessor{batchSize: batchSize, timeout: timeout}
}

func (s *StreamProcessor) ProcessItems(ctx context.Context, items <-chan interface{}, processor func(interface{}) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item, ok := <-items:
			if !ok {
				return nil
			}
			if err := processor(item); err != nil {
				return err
			}
		}
	}
}

type BatchProcessor struct {
	batchSize int
	items     []interface{}
	mu        sync.Mutex
}

func NewBatchProcessor(batchSize int) *BatchProcessor {
	if batchSize <= 0 {
		batchSize = 100
	}
	return &BatchProcessor{batchSize: batchSize, items: make([]interface{}, 0, batchSize)}
}

func (b *BatchProcessor) Add(item interface{}) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.items = append(b.items, item)
	return len(b.items) >= b.batchSize
}

func (b *BatchProcessor) Flush() []interface{} {
	b.mu.Lock()
	defer b.mu.Unlock()
	items := b.items
	b.items = make([]interface{}, 0, b.batchSize)
	return items
}

func (b *BatchProcessor) Size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.items)
}

type StreamingResponse struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	Index   int         `json:"index"`
	Total   int         `json:"total"`
	Message string      `json:"message"`
}

func NewStreamDataResponse(data interface{}, index, total int) *StreamingResponse {
	return &StreamingResponse{Type: "data", Data: data, Index: index, Total: total}
}

func NewStreamErrorResponse(message string) *StreamingResponse {
	return &StreamingResponse{Type: "error", Message: message}
}

func NewStreamDoneResponse(total int) *StreamingResponse {
	return &StreamingResponse{Type: "done", Total: total, Message: "completed"}
}

type PaginatedResult struct {
	Items      interface{} `json:"items"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
	HasMore    bool        `json:"has_more"`
}

func NewPaginatedResult(items interface{}, total, page, pageSize int) *PaginatedResult {
	totalPages := (total + pageSize - 1) / pageSize
	return &PaginatedResult{Items: items, Total: total, Page: page, PageSize: pageSize, TotalPages: totalPages, HasMore: page < totalPages}
}

type Paginator struct {
	DefaultPageSize int
	MaxPageSize     int
}

func NewPaginator(defaultPageSize, maxPageSize int) *Paginator {
	if defaultPageSize <= 0 {
		defaultPageSize = 20
	}
	if maxPageSize <= 0 {
		maxPageSize = 100
	}
	return &Paginator{DefaultPageSize: defaultPageSize, MaxPageSize: maxPageSize}
}

func (p *Paginator) GetPageParams(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = p.DefaultPageSize
	}
	if pageSize > p.MaxPageSize {
		pageSize = p.MaxPageSize
	}
	return page, pageSize
}

func (p *Paginator) GetOffset(page, pageSize int) int {
	page, pageSize = p.GetPageParams(page, pageSize)
	return (page - 1) * pageSize
}

func (p *Paginator) Paginate(items []interface{}, page, pageSize int) *PaginatedResult {
	page, pageSize = p.GetPageParams(page, pageSize)
	total := len(items)
	start := (page - 1) * pageSize
	if start >= total {
		return NewPaginatedResult([]interface{}{}, total, page, pageSize)
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return NewPaginatedResult(items[start:end], total, page, pageSize)
}
