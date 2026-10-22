//go:build !darwin

package main

import (
	"context"
	"encoding/base64"
	"math"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func renderPDF(htmlContent string) ([]byte, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	dataURI := "data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(htmlContent))

	var pdfBuf []byte
	if err := chromedp.Run(ctx,
		chromedp.Navigate(dataURI),
		// Mermaid 렌더링 대기
		chromedp.Sleep(2*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPaperWidth(8.27).   // A4
				WithPaperHeight(11.69). // A4
				WithMarginTop(0.6).
				WithMarginBottom(0.6).
				WithMarginLeft(0.6).
				WithMarginRight(0.6).
				WithPrintBackground(true).
				WithScale(math.Round(1.0*100) / 100).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBuf = buf
			return nil
		}),
	); err != nil {
		return nil, err
	}

	return pdfBuf, nil
}
