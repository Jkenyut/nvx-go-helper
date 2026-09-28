# File Utilities (`/fileutil`)

Secure file validation and handling utilities: MIME type detection via magic bytes, Microsoft Office OLE & OOXML validation, and path traversal sanitization.

## 🚀 Key Features

- **Path Traversal Protection**: `SanitizeFileName` strips directory components (`../`, `..\`) and dangerous characters, preventing arbitrary file overwrite attacks.
- **Deep Magic Bytes Inspection**: Validates genuine file content via first 512 bytes rather than trusting untrusted user file extensions.
- **Compound & OpenXML Office Support**: Deep checks for legacy CFBF (`.doc`, `.xls`, `.ppt`) and modern zipped OOXML (`.docx`, `.xlsx`, `.pptx`).

---

## 🛡️ File Security Inspection Flow

```mermaid
flowchart TD
    FileIn([Upload File Bytes / Filename]) --> ActionChoice{Validation Action}

    ActionChoice -- Sanitize Name --> CleanSlashes[Replace Windows \\ to /]
    CleanSlashes --> BaseName[Extract filepath.Base]
    BaseName --> RegexClean[Remove characters outside a-z, 0-9, ., -, _]
    RegexClean --> TrimDots[Strip leading dots]
    TrimDots --> SafeName([Return Safe Filename])

    ActionChoice -- Check Safe Image --> MagicImg[GetMimeType first 512 bytes]
    MagicImg --> ImageMatch{Match PNG, JPEG, GIF, WebP?}
    ImageMatch -- Yes --> SafeImgTrue([Return true])
    ImageMatch -- No --> SafeImgFalse([Return false])

    ActionChoice -- Check Safe Document --> CheckCFBF{Has CFBF Magic Bytes \xD0\xCF\x11...?}
    CheckCFBF -- Yes --> SafeDocTrue([Return true - Legacy Office])
    CheckCFBF -- No --> CheckZipOOXML{Is ZIP with [Content_Types].xml?}
    CheckZipOOXML -- Yes --> SafeDocTrue
    CheckZipOOXML -- No --> CheckStandardDoc{MIME is PDF, TXT, CSV, RTF?}
    CheckStandardDoc -- Yes --> SafeDocTrue
    CheckStandardDoc -- No --> SafeDocFalse([Return false])
```

---

## 📖 Quickstart & Examples

```go
import "github.com/Jkenyut/nvx-go-helper/fileutil"

// 1. Path traversal sanitization
cleanName := fileutil.SanitizeFileName("../../etc/passwd.png")
// Result: "passwd.png"

// 2. MIME type magic bytes detection
mimeType := fileutil.GetMimeType(fileBytes) // "image/png"

// 3. Security checks
isImage := fileutil.IsSafeImage(fileBytes)
isDoc := fileutil.IsSafeDocument(fileBytes)
```
