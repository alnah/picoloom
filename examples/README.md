# Examples

Sample Markdown files and their generated PDFs.

## Styles Gallery

| Style | Simple | Full-featured |
|-------|--------|---------------|
| academic | `simple-academic.pdf` | `full-academic.pdf` |
| corporate | `simple-corporate.pdf` | `full-corporate.pdf` |
| creative | `simple-creative.pdf` | `full-creative.pdf` |
| default | `simple-default.pdf` | `full-default.pdf` |
| invoice | `simple-invoice.pdf` | `full-invoice.pdf` |
| legal | `simple-legal.pdf` | `full-legal.pdf` |
| manuscript | `simple-manuscript.pdf` | `full-manuscript.pdf` |
| technical | `simple-technical.pdf` | `full-technical.pdf` |

**Simple:** Basic document with tables, code blocks, task lists.
**Full-featured:** Cover page, TOC, signature block, SECRET watermark, footer with page numbers and doc ID, admonitions (all five types, nested).

## Source Files

| File | Description |
|------|-------------|
| `simple-report.md` | Project status report |
| `full-featured.md` | API Security Audit Report |
| `full-featured.yaml` | Config: cover, TOC, signature, watermark, footer |

## Regenerate

```bash
# Simple reports
for style in academic corporate creative default invoice legal manuscript technical; do
  picoloom convert examples/simple-report.md -o "examples/simple-${style}.pdf" --style "$style"
done

# Full-featured reports
for style in academic corporate creative default invoice legal manuscript technical; do
  picoloom convert examples/full-featured.md -o "examples/full-${style}.pdf" \
    -c examples/full-featured.yaml --style "$style"
done
```
