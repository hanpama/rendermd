package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

const htmlTemplate = `<!DOCTYPE html>
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <style>
        /* ── Print ── */
        @media print {
            body { margin: 0; }
            @page { margin: 1.8cm 1.6cm; size: A4; }
            hr { break-after: avoid; }
            h1, h2, h3, h4 { break-after: avoid; }
            table, pre, blockquote { break-inside: avoid; }
        }

        /* ── Base ── */
        body {
            font-family: 'Noto Serif KR', 'Charter', 'Bitstream Charter', 'Iowan Old Style', Georgia, serif;
            font-size: 7.2pt;
            line-height: 1.75;
            color: #1a1a1a;
            max-width: 780px;
            margin: 0 auto;
            padding: 32px 24px 48px;
            -webkit-font-smoothing: antialiased;
            word-break: keep-all;
            overflow-wrap: break-word;
        }

        /* ── Headings ── */
        h1, h2, h3, h4, h5, h6 {
            font-weight: 700;
            letter-spacing: -0.01em;
            color: #0d0d0d;
        }

        h1 {
            font-size: 11.2pt;
            line-height: 1.35;
            margin: 0 0 0.25em 0;
            padding-bottom: 0.45em;
            border-bottom: 1px solid #0d0d0d;
        }

        h2 {
            font-size: 8.8pt;
            margin: 2.4em 0 0.7em 0;
            padding-bottom: 0.3em;
            border-bottom: 1px solid #d0d0d0;
        }

        h3 {
            font-size: 8pt;
            margin: 1.8em 0 0.5em 0;
        }

        h4 {
            font-size: 7.2pt;
            margin: 1.4em 0 0.4em 0;
        }

        /* Reduce top margin when heading immediately follows hr */
        hr + h2 { margin-top: 1.4em; }

        /* ── Body text ── */
        p {
            margin: 0.6em 0;
        }

        /* Tighter spacing between consecutive paragraphs */
        p + p {
            margin-top: 0.35em;
        }

        /* ── Lists ── */
        ul, ol {
            margin: 0.5em 0 0.7em 0;
            padding-left: 1.4em;
        }

        li {
            margin: 0.25em 0;
            line-height: 1.7;
        }

        li > ul, li > ol {
            margin: 0.15em 0 0.15em 0;
        }

        /* ── Inline code ── */
        code {
            font-family: 'IBM Plex Mono', 'JetBrains Mono', 'SF Mono', 'Consolas', monospace;
            font-size: 6.4pt;
            background-color: #f3f4f6;
            border: 1px solid #e5e7eb;
            padding: 0.5px 3.5px;
            word-break: break-all;
            color: #1a1a1a;
        }

        /* ── Code blocks ── */
        pre {
            font-family: 'IBM Plex Mono', 'JetBrains Mono', 'SF Mono', 'Consolas', monospace;
            font-size: 5.6pt;
            line-height: 1.55;
            background-color: #f8f9fa;
            border: 1px solid #e8e8e8;
            border-radius: 3px;
            padding: 1em 1.4em;
            overflow-x: auto;
            margin: 0.9em 0 1.1em 0;
            white-space: pre-wrap;
            word-break: break-all;
            overflow-wrap: anywhere;
        }

        pre code {
            background-color: transparent;
            border: none;
            border-radius: 0;
            padding: 0;
            white-space: pre-wrap;
            font-size: inherit;
        }

        /* ── Blockquote ── */
        blockquote {
            margin: 0.8em 0;
            padding: 0.3em 0 0.3em 1em;
            border-left: 2.5px solid #b0b0b0;
            color: #444;
        }

        blockquote p {
            margin: 0.3em 0;
        }

        /* ── Tables ── */
        table {
            border-collapse: collapse;
            margin: 0.9em 0 1.6em 0;
            font-size: 6.4pt;
            width: 100%;
            line-height: 1.6;
        }

        th, td {
            padding: 0.45em 0.7em;
            border: none;
            border-bottom: 1px solid #e0e0e0;
            text-align: left;
            vertical-align: top;
            overflow-wrap: break-word;
        }

        th {
            font-weight: 700;
            font-size: 6.4pt;
            text-transform: uppercase;
            letter-spacing: 0.04em;
            color: #555;
            border-bottom: 1.5px solid #999;
            padding-bottom: 0.5em;
        }

        /* Subtle zebra stripe */
        tbody tr:nth-child(even) {
            background-color: #fafafa;
        }

        /* First column emphasis for key-value style tables */
        td:first-child {
            white-space: nowrap;
        }

        /* ── Horizontal rule ── */
        hr {
            border: none;
            border-top: 1px solid #e0e0e0;
            margin: 2em 0 1em 0;
        }

        /* ── Links ── */
        a {
            color: #1a1a1a;
            text-decoration: underline;
            text-decoration-color: #aaa;
            text-underline-offset: 1.5px;
        }

        a:hover {
            text-decoration-color: #1a1a1a;
        }

        /* ── Emphasis ── */
        strong {
            font-weight: 700;
            color: #0d0d0d;
        }

        em {
            font-style: italic;
        }

        /* ── Images ── */
        img {
            max-width: 100%;
            height: auto;
        }

        /* ── Mermaid ── */
        .mermaid {
            text-align: center;
            margin: 1em 0;
            max-height: 400px;
            overflow: auto;
        }

        .mermaid svg {
            max-height: 400px;
            width: auto;
        }
    </style>
</head>
<body>
    {{.Content}}
    <script src="https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js"></script>
    <script>
        document.querySelectorAll('pre > code.language-mermaid').forEach(function(el) {
            var pre = el.parentElement;
            var div = document.createElement('div');
            div.className = 'mermaid';
            div.textContent = el.textContent;
            pre.parentElement.replaceChild(div, pre);
        });
        mermaid.initialize({ startOnLoad: true, theme: 'neutral' });
    </script>
</body>
</html>`

type PageData struct {
	Title   string
	Content template.HTML
}

func main() {
	var (
		input  = flag.String("i", "", "Input markdown file")
		output = flag.String("o", "", "Output PDF file (default: input_name.pdf)")
		browse = flag.Bool("b", false, "Open the rendered PDF")
	)

	flag.Parse()

	if *input == "" {
		if flag.NArg() > 0 {
			*input = flag.Arg(0)
		} else {
			fmt.Println("Usage: rendermd -i input.md [-o output.pdf] [-b]")
			fmt.Println("   or: rendermd input.md [output.pdf]")
			fmt.Println("\nOptions:")
			fmt.Println("  -b    Open the rendered PDF")
			os.Exit(1)
		}
	}

	if *output == "" {
		if flag.NArg() > 1 {
			*output = flag.Arg(1)
		} else {
			base := strings.TrimSuffix(filepath.Base(*input), filepath.Ext(*input))
			*output = base + ".pdf"
		}
	}

	content, err := os.ReadFile(*input)
	if err != nil {
		log.Fatalf("Failed to read input file: %v", err)
	}

	htmlContent, err := renderMarkdown(content)
	if err != nil {
		log.Fatalf("Failed to render markdown: %v", err)
	}

	title := strings.TrimSuffix(filepath.Base(*input), filepath.Ext(*input))

	htmlStr, err := buildHTML(htmlContent, title)
	if err != nil {
		log.Fatalf("Failed to build HTML: %v", err)
	}

	pdfBuf, err := renderPDF(htmlStr)
	if err != nil {
		log.Fatalf("Failed to render PDF: %v", err)
	}

	if err := os.WriteFile(*output, pdfBuf, 0644); err != nil {
		log.Fatalf("Failed to write PDF: %v", err)
	}

	fmt.Printf("Successfully rendered %s to %s\n", *input, *output)

	if *browse {
		absPath, err := filepath.Abs(*output)
		if err != nil {
			log.Fatalf("Failed to get absolute path: %v", err)
		}
		if err := openFile(absPath); err != nil {
			log.Fatalf("Failed to open file: %v", err)
		}
		fmt.Println("Opened PDF")
	}
}

func renderMarkdown(source []byte) ([]byte, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.GFM,
			extension.DefinitionList,
			extension.Footnote,
			extension.Typographer,
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func buildHTML(content []byte, title string) (string, error) {
	tmpl, err := template.New("page").Parse(htmlTemplate)
	if err != nil {
		return "", err
	}

	data := PageData{
		Title:   title,
		Content: template.HTML(content),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

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

func openFile(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", path)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
