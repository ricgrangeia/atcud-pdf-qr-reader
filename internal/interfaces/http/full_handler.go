package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	appConfig "cmd/go-api/internal/config"
	"cmd/go-api/internal/infrastructure/stats"
)

// FullInvoiceBody is the merged response of /api/v1/document/full.
// `invoice` contains the full invoice extraction (QR + payment data, headers, totals,
// seller, buyer, dates, etc.). `items` contains only the line-items table.
type FullInvoiceBody struct {
	Invoice json.RawMessage `json:"invoice" doc:"Dados completos da fatura (cabeçalhos, totais, vendedor, comprador) extraídos via /tools/pdf/invoice/decode-upload"`
	Items   ItemsBody       `json:"items"   doc:"Linhas da tabela de itens extraídas via /tools/pdf/items/decode-upload"`
}

// FullInvoiceOutput wraps FullInvoiceBody for Huma.
type FullInvoiceOutput struct {
	Body FullInvoiceBody
}

// FullInvoiceHandler proxies a PDF to the tool server's invoice + items extractors in
// parallel and returns a merged JSON response.
func FullInvoiceHandler(cfg *appConfig.Config, counter *stats.Counter) func(context.Context, *ParseInput) (*FullInvoiceOutput, error) {
	return func(ctx context.Context, input *ParseInput) (*FullInvoiceOutput, error) {
		if cfg.ToolServerURL == "" {
			return nil, huma.Error503ServiceUnavailable("full invoice extraction requires TOOL_SERVER_URL to be configured")
		}

		pdfBytes, err := io.ReadAll(input.RawBody.Data().File)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("could not read the uploaded file", fmt.Errorf("io.ReadAll: %w", err))
		}

		type result struct {
			kind string
			data []byte
			err  error
		}
		ch := make(chan result, 2)

		go func() {
			data, err := callToolServerUpload(ctx, cfg, "/tools/pdf/invoice/decode-upload", pdfBytes)
			ch <- result{kind: "invoice", data: data, err: err}
		}()
		go func() {
			data, err := callToolServerUpload(ctx, cfg, "/tools/pdf/items/decode-upload", pdfBytes)
			ch <- result{kind: "items", data: data, err: err}
		}()

		var invoiceData, itemsData []byte
		for i := 0; i < 2; i++ {
			r := <-ch
			if r.err != nil {
				return nil, huma.Error502BadGateway("calling tool server "+r.kind+" endpoint", r.err)
			}
			if r.kind == "invoice" {
				invoiceData = r.data
			} else {
				itemsData = r.data
			}
		}

		// items: only keep columns + rows.
		var itemsWrapper struct {
			Items struct {
				Columns []string                 `json:"columns"`
				Rows    []map[string]interface{} `json:"rows"`
			} `json:"items"`
		}
		if err := json.Unmarshal(itemsData, &itemsWrapper); err != nil {
			return nil, huma.Error502BadGateway("parsing items response", err)
		}

		counter.Increment(sourceFromContext(ctx))

		return &FullInvoiceOutput{Body: FullInvoiceBody{
			Invoice: invoiceData,
			Items: ItemsBody{
				Columns: itemsWrapper.Items.Columns,
				Rows:    itemsWrapper.Items.Rows,
			},
		}}, nil
	}
}

// callToolServerUpload posts a PDF as multipart/form-data to the given tool-server path.
// Uses the shared itemsClient (5-minute timeout).
func callToolServerUpload(ctx context.Context, cfg *appConfig.Config, path string, pdfBytes []byte) ([]byte, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "document.pdf")
	if err != nil {
		return nil, fmt.Errorf("creating multipart form: %w", err)
	}
	if _, err = part.Write(pdfBytes); err != nil {
		return nil, fmt.Errorf("writing PDF to form: %w", err)
	}
	w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.ToolServerURL+path, &body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if cfg.ToolServerAPIKey != "" {
		req.Header.Set("x-api-key", cfg.ToolServerAPIKey)
	}

	resp, err := itemsClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling %s: %w", path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading %s response: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned HTTP %d: %s", path, resp.StatusCode, string(data))
	}
	return data, nil
}
