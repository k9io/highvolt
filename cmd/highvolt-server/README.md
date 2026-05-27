

        // Images depend on the pre-processor (i.e - ollama),  but should support....
        // "When converting Word documents for AI analysis, text density matters. A standard Word page
        // converted at 72 DPI (the default) might be too blurry for the model to read small footnotes.
        // I recommend rendering at 150 or 200 DPI. This creates a sharp enough image for Ministral-3 to
        // perform high-accuracy OCR without making the file size so large that it slows down your GPU."
        //
        // Images Supported:
        //
        // PNG, JPEG / JPG, WebP, BMP, TIFF, HEIC
        //
        // soffice libreoffice / headless
        //
        // Images: PNG, JPEG / JPG, WebP, BMP, TIFF, HEIC
        // Text: .doc, .docx, .rtf, .odt, .txt, .pages, .wpd
        // Spreadsheets: .xls, .xlsx, .csv, .ods, .numbers
        // Presentations: .ppt, .pptx, .odp, .key
        // Graphics/Drawings: .vsd (Visio), .cdr (CorelDRAW), .pub (Publisher), .svg

        // Future compression formats: https://github.com/mholt/archives
        // brotli (.br), bzip2 (.bz2), flate (.zip), gzip (.gz), lz4 (.lz4), lzip (.lz)
        // minlz (.mz), snappy (.sz) and S2 (.s2), xz (.xz), zlib (.zz), zstandard (.zst),

        // .zip, .tar (including any compressed variants like .tar.gz), .rar (read-only), .7z (read-only)

        /* Images - When using Visual AI models,  these should be natively supported */

