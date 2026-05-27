# File Processing Pipeline

When a submission is dequeued by a worker, the MIME type is classified and routed to one of five processing paths.

## MIME classification

The `helpers.Check_MIME` function compares the submitted `mimetype` field against five lists from the server configuration:

| Category | Return token | Processor |
|----------|-------------|-----------|
| `core.mime_types.image` | `IMAGE` | Direct LLM vision request |
| `core.mime_types.pdf` | `PDF` | PDF → PNG pages → LLM |
| `core.mime_types.office` | `OFFICE` | Office → PDF → PNG → LLM |
| `core.mime_types.archive` | `ARCHIVE` | Extract → classify each file → recurse |
| `core.mime_types.text` | `TEXT` | Decode base64 → LLM text request |

Files with a MIME type not in any list are logged and dropped.

---

## IMAGE processor

Images are sent directly to the LLM as a vision request.

- Images below `core.minimum_image_size` bytes (decoded) are rejected — too small to contain readable PII.
- The base64 payload is wrapped in an OpenAI-format `image_url` content part.

---

## TEXT processor

Text files (plain text, CSV, etc.) are decoded from base64 and appended to the user prompt before being sent as a standard chat completion request.

---

## PDF processor

PDFs cannot be sent directly to most vision LLMs. The pipeline:

1. Decodes the base64 PDF and writes it to a temp directory.
2. Runs the configured `export_commands.pdf` command (e.g. `pdftoppm`) to convert each page to a PNG image.
3. Sorts the pages with natural-sort order.
4. Submits each page image to the LLM in sequence.
5. **Stops at the first page that contains sensitive data** — returns that page's result along with the `page_number`.
6. If no page triggers a positive result, returns the first page's verdict (clean).
7. Stops after `core.max_pdf_pages` pages regardless, to bound resource use on large documents.

The temp directory is always cleaned up (`defer os.RemoveAll`) regardless of outcome.

### PDF command placeholders

| Placeholder | Value |
|-------------|-------|
| `%INFILE%` | Path to the decoded PDF temp file |
| `%WORKDIR%` | Path to the temp working directory |
| `%OUTFILE%` | Output filename pattern (e.g. `highvolt-pdf-%d.png`) |
| `%RANGE%` | Page range (e.g. `[0-49]` for max 50 pages) |

Example command:
```
pdftoppm -png -r 150 %INFILE% %WORKDIR%/%OUTFILE%
```

---

## OFFICE processor

Microsoft Office formats (docx, xlsx, pptx, etc.) cannot be decoded by Highvolt natively. The pipeline delegates to LibreOffice:

1. Decodes base64 and writes the file to a temp directory.
2. Runs the configured `export_commands.office` command to convert the document to PDF.
3. Passes the resulting PDF to the **PDF processor**.

Pipeline: Office → PDF → PNG pages → LLM.

### Office command placeholders

| Placeholder | Value |
|-------------|-------|
| `%INFILE%` | Path to the decoded Office document |
| `%WORKDIR%` | Temp working directory |
| `%OUTFILE%` | Output filename pattern |

Example command:
```
libreoffice --headless --convert-to pdf --outdir %WORKDIR% %INFILE%
```

---

## ARCHIVE processor

Archives (ZIP, tar.gz, etc.) are extracted and their contents analyzed recursively.

1. Decodes base64 and writes the archive to a temp directory.
2. Identifies the archive format using the `mholt/archives` library.
3. Extracts all files, enforcing:
   - **Path traversal protection**: any `NameInArchive` that escapes the work directory is skipped.
   - **Zip bomb protection**: total extracted bytes are tracked atomically; extraction stops if `core.max_archive_size` is exceeded.
   - **Extraction timeout**: the entire extraction must complete within `core.archive_extract_timeout` seconds.
4. Walks the extracted files with natural-sort order.
5. For each file, classifies its MIME type (using magic bytes), and calls `Submit_Data` recursively.
6. **Stops at the first file with sensitive data** — returns that result with the relative `file_in_zip` path added.
7. If all files are clean, returns a hard-coded clean verdict.
