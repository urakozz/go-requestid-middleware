package requestid

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/Home24/Base-Go/Godeps/_workspace/src/github.com/codegangsta/negroni"
	"fmt"
	"time"
	"github.com/gorilla/context"
)

const (
	// DefaultIDHeader is where command ID is stored.
	DefaultIDHeader = "X-Command-ID"
)


type (
// IDGenerator interface for various implementations
// to generate request ID
	IDGenerator interface {
		Generate() string
	}

	// IDSource (E.g.: header, custom)
	IDSource interface {
		GetID(r *http.Request) string
	}

	// IDSaveHandler (E.g.: header, context, custom)
	IDSaveHandler interface {
		SaveID(rw http.ResponseWriter, r *http.Request, id string)
	}

	// IDPostProcessor (E.g.: header, custom)
	IDPostProcessor interface {
		Process(rw http.ResponseWriter, r *http.Request, id string)
	}

	randomIDGenerator struct{}

	timestampIDGenerator struct{}

)

// RequestIDInjector injects random command ID into request and response.
type RequestIDInjector interface {
	negroni.Handler
}

// IDInjectorOptions contains options for initialization requestIDInjector
type IDInjectorOptions struct {
	IDGenerator IDGenerator
	IDSource IDSource
	IDSaveHandler IDSaveHandler
	IDPostProcessor IDPostProcessor
}

// NewRequestIDInjector does what it says on the tin.
// Usage:
//  app := negroni.New()
//  app.Use(requestid.NewRequestIDInjector(&requestid.IDInjectorOptions{}))
//  return context.ClearHandler(app)
// Is the same as:
//  app := negroni.New()
//  app.Use(requestid.NewRequestIDInjector(&requestid.IDInjectorOptions{
//    IDGenerator :     NewTimestampIDGenerator(),
//    IDSource :        requestid.NewSourceHeader(requestid.DefaultIDHeader),
//    IDSaveHandler :   requestid.NewSaveHandlerHeader(requestid.DefaultIDHeader),
//    IDPostProcessor : requestid.NewPostProcessorHeader(requestid.DefaultIDHeader),
//  }))
//  return context.ClearHandler(app)
func NewRequestIDInjector(o *IDInjectorOptions) RequestIDInjector {
	middleware := &requestIDInjector{o.IDGenerator, o.IDSource, o.IDSaveHandler, o.IDPostProcessor}
	middleware.applyDefaults()
	return middleware
}

// GetRequestID extracts command ID from the request header.
func GetRequestID(r *http.Request) string {
	return r.Header.Get(DefaultIDHeader)
}


// Handlers

type requestIDInjector struct {
	idGenerator IDGenerator
	idSource IDSource
	idSaveHandler IDSaveHandler
	idPostProcessor IDPostProcessor
}

func (v *requestIDInjector) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	id := v.idSource.GetID(r)
	if id == "" {
		id = v.idGenerator.Generate()
	}
	v.idSaveHandler.SaveID(rw, r, id)
	v.idPostProcessor.Process(rw, r, id)

	next(rw, r)
}

func (v *requestIDInjector) applyDefaults(){
	if v.idGenerator == nil {
		v.idGenerator = NewTimestampIDGenerator()
	}
	if v.idSource == nil {
		v.idSource = NewSourceHeader(DefaultIDHeader)
	}
	if v.idSaveHandler == nil {
		v.idSaveHandler = NewSaveHandlerHeader(DefaultIDHeader)
	}
	if v.idPostProcessor == nil {
		v.idPostProcessor  = NewPostProcessorHeader(DefaultIDHeader)
	}
}

// NewRandomIDGenerator does what it says on the tin.
func NewRandomIDGenerator() IDGenerator {
	return &randomIDGenerator{}
}

// NewTimestampIDGenerator does what it says on the tin.
func NewTimestampIDGenerator() IDGenerator {
	return &timestampIDGenerator{}
}

// Generate random id
func (rig *randomIDGenerator) Generate() string {
	r := make([]byte, 16)
	if _, err := rand.Read(r); err != nil {
		return ""
	}

	return hex.EncodeToString(r)
}

// Generate id based on time.Now().UnixNano()
func (trg *timestampIDGenerator) Generate() string {
	r := make([]byte, 2)
	rand.Read(r)
	id := fmt.Sprintf("%f.%s", float64(time.Now().UnixNano())/1000000000, hex.EncodeToString(r))

	return id
}

// Source Header
type sourceHeader struct{
	header string
}

func (s *sourceHeader) GetID(r *http.Request) string {
	return r.Header.Get(s.header)
}

// NewSourceHeader returns new sourceHeader
func NewSourceHeader(header string) IDSource {
	return &sourceHeader{header}
}

// Source Custom
type sourceCustom struct{
	fn func(r *http.Request) string
}

func (s *sourceCustom) GetID(r *http.Request) string {
	return s.fn(r)
}

// NewSourceCustom returns new sourceCustom
func NewSourceCustom(fn func(r *http.Request) string) IDSource {
	return &sourceCustom{fn}
}

//SaveHandlerHeader
type saveHandlerHeader struct{
	header string
}

func (s *saveHandlerHeader) SaveID(rw http.ResponseWriter, r *http.Request, id string) {
	r.Header.Set(s.header, id)
}
// NewSaveHandlerHeader returns new saveHandlerHeader (IDSaveHandler interface)
func NewSaveHandlerHeader(header string) IDSaveHandler {
	return &saveHandlerHeader{header}
}

//SaveHandlerContext
type saveHandlerContext struct{
	field interface{}
}

func (s *saveHandlerContext) SaveID(rw http.ResponseWriter, r *http.Request, id string) {
	context.Set(r, s.field, id)
}

// NewSaveHandlerContext returns new saveHandlerContext (IDSaveHandler interface)
func NewSaveHandlerContext(field interface{}) IDSaveHandler {
	return &saveHandlerContext{field}
}

// SaveHandler Custom
type saveHandlerCustom struct{
	fn func(rw http.ResponseWriter, r *http.Request, id string)
}

func (s *saveHandlerCustom) SaveID(rw http.ResponseWriter, r *http.Request, id string) {
	s.fn(rw, r, id)
}

// NewSaveHandlerCustom returns new saveHandlerCustom (IDSaveHandler interface)
func NewSaveHandlerCustom(fn func(rw http.ResponseWriter, r *http.Request, id string)) IDSaveHandler {
	return &saveHandlerCustom{fn}
}

//IDPostProcessor Header
type postProcessorHeader struct {
	header string
}
func (p *postProcessorHeader) Process(rw http.ResponseWriter, r *http.Request, id string) {
	rw.Header()[p.header] = []string{id}
}
// NewPostProcessorHeader returns new postProcessorHeader (IDPostProcessor interface)
func NewPostProcessorHeader(field string) IDPostProcessor {
	return &postProcessorHeader{field}
}

//IDPostProcessor Custom
type postProcessorCustom struct {
	fn func(rw http.ResponseWriter, r *http.Request, id string)
}
func (p *postProcessorCustom) Process(rw http.ResponseWriter, r *http.Request, id string) {
	p.fn(rw, r, id)
}
// NewPostProcessorCustom returns new postProcessorCustom (IDPostProcessor interface)
func NewPostProcessorCustom(fn func(rw http.ResponseWriter, r *http.Request, id string)) IDPostProcessor {
	return &postProcessorCustom{fn}
}


