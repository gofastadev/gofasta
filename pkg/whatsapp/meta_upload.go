package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

// multipartBodyBufFn produces the buffer that multipart.NewWriter
// renders into. Var (not const) so tests can substitute a writer
// that fails on demand, exercising the rare `if err != nil` branches
// after each Write that are otherwise unreachable when writing to a
// bytes.Buffer (which never fails). Production always returns a
// fresh *bytes.Buffer.
type readWriteBuffer interface {
	io.Reader
	io.Writer
}

var multipartBodyBufFn = func() readWriteBuffer { return &bytes.Buffer{} }

// uploadMetaMedia POSTs raw bytes to /{phone-number-id}/media and
// returns the resulting media ID. Split out so the main meta.go file
// stays focused on the message-send path.
func uploadMetaMedia(ctx context.Context, client *http.Client, endpoint, accessToken string, media *MediaAttachment) (string, error) {
	if media.ContentType == "" {
		return "", errMissingContentType
	}

	body := multipartBodyBufFn()
	mw := multipart.NewWriter(body)
	if writeErr := mw.WriteField("messaging_product", "whatsapp"); writeErr != nil {
		return "", writeErr
	}
	if writeErr := mw.WriteField("type", media.ContentType); writeErr != nil {
		return "", writeErr
	}
	filename := media.Filename
	if filename == "" {
		filename = "upload"
	}
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", fmt.Sprintf("form-data; name=\"file\"; filename=%q", filename))
	hdr.Set("Content-Type", media.ContentType)
	part, partErr := mw.CreatePart(hdr)
	if partErr != nil {
		return "", partErr
	}
	if _, writeErr := part.Write(media.Content); writeErr != nil {
		return "", writeErr
	}
	if closeErr := mw.Close(); closeErr != nil {
		return "", closeErr
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("meta media %d: %s", resp.StatusCode, string(respBytes))
	}
	var u struct {
		ID string `json:"id"`
	}
	if jsonErr := json.Unmarshal(respBytes, &u); jsonErr != nil {
		return "", fmt.Errorf("meta media decode: %w (body: %s)", jsonErr, string(respBytes))
	}
	if u.ID == "" {
		return "", fmt.Errorf("meta media: empty id (body: %s)", string(respBytes))
	}
	return u.ID, nil
}
