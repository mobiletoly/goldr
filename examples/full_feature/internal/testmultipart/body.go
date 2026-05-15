package testmultipart

import (
	"bytes"
	"mime/multipart"
)

type Upload struct {
	Filename string
	Content  string
}

type TB interface {
	Helper()
	Fatalf(format string, args ...any)
}

func Body(t TB, fields map[string]string, files map[string]Upload) (*bytes.Buffer, string) {
	t.Helper()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatalf("WriteField() error = %v", err)
		}
	}
	for name, upload := range files {
		part, err := writer.CreateFormFile(name, upload.Filename)
		if err != nil {
			t.Fatalf("CreateFormFile() error = %v", err)
		}
		if _, err := part.Write([]byte(upload.Content)); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return body, writer.FormDataContentType()
}
