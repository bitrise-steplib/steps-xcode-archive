package utils

import (
	"encoding/json"
	"io"
)

//go:generate moq -out mocks/encoder_mock.go -pkg mocks . Encoder
type Encoder interface {
	SetIndent(prefix, indent string)
	SetEscapeHTML(escape bool)
	Encode(data any) error
}

//go:generate moq -out mocks/encoder_factory_mock.go -pkg mocks . EncoderFactory
type EncoderFactory interface {
	Encoder(w io.Writer) Encoder
}

//go:generate moq -out mocks/decoder_mock.go -pkg mocks . Decoder
type Decoder interface {
	Decode(data any) error
}

//go:generate moq -out mocks/decoder_factory_mock.go -pkg mocks . DecoderFactory
type DecoderFactory interface {
	Decoder(r io.Reader) Decoder
}

type DefaultEncoderFactory struct{}

type DefaultDecoderFactory struct{}

// Intentionally skipping interface return error - we are using this interface in many commands and their tests

func (factory DefaultEncoderFactory) Encoder(w io.Writer) Encoder {
	return json.NewEncoder(w)
}

func (factory DefaultDecoderFactory) Decoder(r io.Reader) Decoder {
	return json.NewDecoder(r)
}
