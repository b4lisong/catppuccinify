# Catppuccinify — Product Requirements Document & Entity Relationship Diagram

## 1. Overview

**Catppuccinify** is a web application that converts any uploaded image's colors to match the [Catppuccin Mocha](https://catppuccin.com/palette/) color palette. It uses perceptually accurate color matching in CIELAB color space with Floyd-Steinberg dithering to produce high-quality, natural-looking results.

The app is a single-purpose tool: upload an image, press a button, get a Catppuccin-themed version back.

---

## 2. User Flow

1. User opens the web page (served by the Go backend).
2. User uploads or drags-and-drops an image onto the drop zone.
3. A thumbnail preview of the uploaded image is displayed.
4. User clicks the **"Catppuccinify!"** button.
5. The frontend sends the image to the backend API, receives a job ID.
6. The frontend polls the job status endpoint every 2 seconds.
7. While processing, the UI shows a loading/progress indicator.
8. On completion, the frontend displays a **side-by-side before/after comparison** of the original and converted images.
9. A **"Download"** button is presented. Clicking it downloads the converted PNG.
10. On failure, an inline error message is displayed (styled in Catppuccin Red `#f38ba8`).

---

## 3. Technology Stack

### Backend
- **Language:** Go (latest stable)
- **HTTP framework:** `net/http` from the standard library (no third-party router needed for 3 endpoints)
- **Key dependencies:**
  - [`lucasb-eyer/go-colorful`](https://github.com/lucasb-eyer/go-colorful) — sRGB ↔ CIELAB color space conversion
  - [`golang.org/x/image/webp`](https://pkg.go.dev/golang.org/x/image/webp) — WebP image decoding
  - Standard library `image`, `image/png`, `image/jpeg` — PNG/JPEG decode and PNG encode
- **No database.** Job state is held in-memory (`sync.Map`), processed files in a temp directory.

### Frontend
- **Plain HTML + CSS + vanilla JavaScript** — no framework, no build step.
- Served as static files by the Go backend.
- The UI itself is themed in **Catppuccin Mocha** colors.

---

## 4. Target Color Palette — Catppuccin Mocha

The palette consists of 26 colors. All hex values below are definitive and must be hardcoded in the Go backend.

### Accent Colors

| Name       | Hex       | RGB                |
|------------|-----------|--------------------|
| Rosewater  | `#f5e0dc` | `rgb(245,224,220)` |
| Flamingo   | `#f2cdcd` | `rgb(242,205,205)` |
| Pink       | `#f5c2e7` | `rgb(245,194,231)` |
| Mauve      | `#cba6f7` | `rgb(203,166,247)` |
| Red        | `#f38ba8` | `rgb(243,139,168)` |
| Maroon     | `#eba0ac` | `rgb(235,160,172)` |
| Peach      | `#fab387` | `rgb(250,179,135)` |
| Yellow     | `#f9e2af` | `rgb(249,226,175)` |
| Green      | `#a6e3a1` | `rgb(166,227,161)` |
| Teal       | `#94e2d5` | `rgb(148,226,213)` |
| Sky        | `#89dceb` | `rgb(137,220,235)` |
| Sapphire   | `#74c7ec` | `rgb(116,199,236)` |
| Blue       | `#89b4fa` | `rgb(137,180,250)` |
| Lavender   | `#b4befe` | `rgb(180,190,254)` |

### Neutral / Surface Colors

| Name      | Hex       | RGB                |
|-----------|-----------|--------------------|
| Text      | `#cdd6f4` | `rgb(205,214,244)` |
| Subtext 1 | `#bac2de` | `rgb(186,194,222)` |
| Subtext 0 | `#a6adc8` | `rgb(166,173,200)` |
| Overlay 2 | `#9399b2` | `rgb(147,153,178)` |
| Overlay 1 | `#7f849c` | `rgb(127,132,156)` |
| Overlay 0 | `#6c7086` | `rgb(108,112,134)` |
| Surface 2 | `#585b70` | `rgb(88,91,112)`   |
| Surface 1 | `#45475a` | `rgb(69,71,90)`    |
| Surface 0 | `#313244` | `rgb(49,50,68)`    |
| Base      | `#1e1e2e` | `rgb(30,30,46)`    |
| Mantle    | `#181825` | `rgb(24,24,37)`    |
| Crust     | `#11111b` | `rgb(17,17,27)`    |

---

## 5. Core Image Processing Algorithm

### 5.1 Overview

The algorithm converts every pixel in the input image to its perceptually nearest color in the Catppuccin Mocha palette, using CIELAB color space for distance calculations and Floyd-Steinberg dithering to reduce posterization artifacts.

### 5.2 Steps

1. **Decode the input image.** Accept PNG, JPEG, or WebP. Decode into a standard Go `image.Image`.
2. **Pre-convert the Mocha palette to CIELAB.** At application startup (or package init), convert all 26 Mocha sRGB colors to CIELAB coordinates using `go-colorful`. Store these as a lookup table. This is done once, not per-request.
3. **Create a mutable working copy** of the image as a floating-point pixel buffer (to handle dithering error diffusion, which can produce intermediate values outside 0-255). A `[][]float64` buffer (or equivalent struct with R, G, B float channels) sized to image dimensions.
4. **Iterate over pixels left-to-right, top-to-bottom** (scanline order, required for Floyd-Steinberg):
   - a. Read the current pixel's (potentially error-adjusted) RGB values from the working buffer.
   - b. Clamp the RGB values to [0, 255].
   - c. Convert the clamped sRGB value to CIELAB using `go-colorful`.
   - d. Compute the Euclidean distance in CIELAB space between this pixel and each of the 26 palette colors. Select the palette color with the smallest distance. (With only 26 colors, brute-force linear search is the correct approach — no spatial index needed.)
   - e. **Compute the quantization error:** `error = original_pixel_RGB - chosen_palette_color_RGB` (per channel, in sRGB space — dithering is applied in sRGB, not CIELAB).
   - f. **Diffuse the error** to neighboring pixels using the Floyd-Steinberg kernel:
     ```
         * 7/16
     3/16 5/16 1/16
     ```
     Where `*` is the current pixel. Add the weighted error to the right neighbor, bottom-left, bottom, and bottom-right pixels in the working buffer. Skip neighbors that fall outside the image boundary.
   - g. Write the chosen palette color's sRGB value to the output image at this pixel position.
5. **Encode the output image as PNG** and write to the temp directory.

### 5.3 Alpha Channel Handling

If the input image has an alpha channel, preserve it as-is. Only the RGB channels are palette-mapped. Fully transparent pixels (alpha = 0) can be skipped entirely for performance.

### 5.4 Performance Notes

- The palette CIELAB lookup table is computed once at startup. Do not recompute per-request.
- For a 3840×2160 image with 26 palette colors, this is ~8.3M pixel × 26 distance calculations — straightforward and fast in Go. No goroutine-level parallelism is required for the pixel loop itself (Floyd-Steinberg is inherently sequential due to error diffusion dependencies). However, multiple concurrent requests are naturally handled by Go's HTTP server goroutines.
- Image decode and encode are the likely bottlenecks, not the palette matching.

---

## 6. API Design

All endpoints are served by the same Go HTTP server. The server also serves the static frontend files.

### 6.1 Endpoints

#### `POST /api/convert`

Upload an image for conversion.

**Request:**
- Content-Type: `multipart/form-data`
- Form field name: `image`
- Max body size: **10 MB** (enforced server-side via `http.MaxBytesReader`)

**Validations:**
- File must be present.
- File size ≤ 10 MB.
- File must be a valid PNG, JPEG, or WebP image (validate by attempting decode, not by file extension or MIME type alone).

**Response (201 Created):**
```json
{
  "job_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
}
```

**Error Responses:**
- `400 Bad Request` — missing file, invalid format, or exceeds size limit.
  ```json
  {
    "error": "File exceeds maximum size of 10 MB"
  }
  ```
- `500 Internal Server Error` — unexpected server failure.
  ```json
  {
    "error": "Internal server error"
  }
  ```

#### `GET /api/status/{job_id}`

Check the status of a conversion job.

**Response (200 OK):**
```json
{
  "job_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "processing"
}
```

Possible `status` values:
- `"pending"` — job queued, not yet started.
- `"processing"` — actively converting.
- `"done"` — conversion complete, ready for download.
- `"failed"` — conversion failed.

When `status` is `"failed"`:
```json
{
  "job_id": "...",
  "status": "failed",
  "error": "Failed to decode image"
}
```

**Error Responses:**
- `404 Not Found` — unknown job ID or job has expired.

#### `GET /api/download/{job_id}`

Download the converted image.

**Response (200 OK):**
- Content-Type: `image/png`
- Content-Disposition: `attachment; filename="catppuccinified.png"`
- Body: raw PNG bytes.

**Error Responses:**
- `404 Not Found` — unknown job ID, expired, or not yet done.
- `409 Conflict` — job exists but is not in `"done"` state.

### 6.2 Static File Serving

- `GET /` and all non-`/api/` routes serve the static frontend from an embedded directory (using `embed.FS` or a `static/` directory).

---

## 7. Job Lifecycle & Cleanup

### 7.1 Job State Machine

```
pending → processing → done
                     → failed
```

### 7.2 In-Memory Job Store

- Jobs are stored in a `sync.Map` keyed by job ID (UUID v4, generated via `google/uuid` or `crypto/rand`).
- Each job entry contains: `id`, `status`, `error` (if failed), `created_at`, and `output_path` (path to the result PNG in the temp directory).
- The original uploaded image is also saved to the temp directory for the before/after comparison (served via the download endpoint or a separate static route).

### 7.3 Temp Directory

- All uploaded originals and processed output files are stored in a temp directory (e.g., `os.MkdirTemp` or a configurable path).
- File naming: `{job_id}_original.{ext}` and `{job_id}_output.png`.

### 7.4 Cleanup

- A background goroutine runs on a ticker (e.g., every 60 seconds).
- It iterates all jobs and removes any where `time.Since(created_at) > 10 minutes`.
- Removal means: delete the job from the `sync.Map`, delete the associated files from the temp directory.
- Cleanup errors are logged but do not crash the server.

---

## 8. Frontend Specification

### 8.1 Technology

- Single `index.html` file containing all HTML, CSS, and JavaScript.
- No frameworks, no build step, no external JS dependencies.
- Served by the Go backend as a static file.

### 8.2 Layout

The page is a single centered column with:
1. **Header:** App name "Catppuccinify" in a large, bold font. Optionally, a small subtitle/tagline (e.g., "Convert any image to the Catppuccin Mocha palette").
2. **Drop zone / upload area:** A dashed-border rectangle. Supports both click-to-browse and drag-and-drop. Shows accepted formats ("PNG, JPEG, or WebP — max 10 MB").
3. **Image preview area:** After a file is selected, shows a thumbnail of the uploaded image (generated client-side via `URL.createObjectURL()`). A "remove/clear" option to deselect.
4. **Action button:** Labeled **"Catppuccinify!"**. Disabled until an image is selected. On click, uploads the image, starts polling, and enters a loading state.
5. **Loading state:** The button shows a spinner or "Processing..." text. The drop zone is disabled during processing.
6. **Result area:** On completion, displays the original and converted images side-by-side. On small screens (mobile), stack vertically. Each image is labeled "Original" and "Catppuccinified".
7. **Download button:** Labeled "Download". Triggers a browser download of the converted PNG.
8. **Error display:** Inline error messages below the drop zone, styled with the Catppuccin Red color. Auto-clear on next upload attempt.

### 8.3 Catppuccin Mocha Theme

The entire UI is styled using the Catppuccin Mocha palette:

| UI Element           | Color        | Hex       |
|----------------------|--------------|-----------|
| Page background      | Base         | `#1e1e2e` |
| Card/container bg    | Mantle       | `#181825` |
| Primary text         | Text         | `#cdd6f4` |
| Secondary text       | Subtext 1    | `#bac2de` |
| Border / dividers    | Surface 0    | `#313244` |
| Drop zone border     | Overlay 0    | `#6c7086` |
| Drop zone hover      | Surface 1    | `#45475a` |
| Primary button bg    | Mauve        | `#cba6f7` |
| Primary button text  | Crust        | `#11111b` |
| Button hover         | Lavender     | `#b4befe` |
| Error text           | Red          | `#f38ba8` |
| Success accent       | Green        | `#a6e3a1` |
| Links                | Sapphire     | `#74c7ec` |

### 8.4 Client-Side Validation

Before uploading, the frontend validates:
- A file is selected.
- File size ≤ 10 MB.
- File type is one of `image/png`, `image/jpeg`, `image/webp` (checked via `File.type`).

Invalid files show an inline error and are not uploaded.

### 8.5 Polling Logic

```
POST /api/convert (multipart form with image)
  → receive { job_id }
  → setInterval every 2000ms:
      GET /api/status/{job_id}
        → if "done": clearInterval, fetch and display results
        → if "failed": clearInterval, show error
        → if "pending" or "processing": continue polling
```

### 8.6 Displaying Results

Once the job is `"done"`:
- Fetch the converted image from `GET /api/download/{job_id}` and display it as an `<img>` (using `URL.createObjectURL` on the response blob).
- Display the original image (already available client-side from the upload preview) alongside the converted image.
- Show the "Download" button, which triggers `<a href="..." download="catppuccinified.png">`.

---

## 9. Project Structure

```
catppuccinify/
├── main.go                 # Entry point, HTTP server setup, static file serving
├── go.mod
├── go.sum
├── internal/
│   ├── api/
│   │   ├── handler.go      # HTTP handlers (convert, status, download)
│   │   └── router.go       # Route registration
│   ├── converter/
│   │   ├── palette.go      # Catppuccin Mocha palette definition (sRGB + CIELAB)
│   │   ├── convert.go      # Core conversion algorithm (CIELAB matching + Floyd-Steinberg)
│   │   └── convert_test.go # Unit tests for the converter
│   └── job/
│       ├── store.go        # In-memory job store (sync.Map wrapper)
│       └── cleanup.go      # Background cleanup goroutine
├── static/
│   └── index.html          # Complete frontend (HTML + CSS + JS in one file)
└── README.md
```

---

## 10. Entity Relationship Diagram

This is an in-memory system with no database, but the data model is still well-defined.

### 10.1 Entities

#### Job

| Field        | Type      | Description                                      |
|--------------|-----------|--------------------------------------------------|
| `ID`         | `string`  | UUID v4, primary key                             |
| `Status`     | `string`  | One of: `pending`, `processing`, `done`, `failed`|
| `Error`      | `string`  | Error message (empty unless status is `failed`)  |
| `CreatedAt`  | `time.Time` | Timestamp of job creation                      |
| `InputPath`  | `string`  | Filesystem path to the uploaded original image   |
| `OutputPath` | `string`  | Filesystem path to the converted PNG             |
| `InputName`  | `string`  | Original filename (for reference)                |

#### PaletteColor

| Field   | Type      | Description                                  |
|---------|-----------|----------------------------------------------|
| `Name`  | `string`  | Human-readable name (e.g., "Rosewater")      |
| `Hex`   | `string`  | Hex color code                               |
| `R`     | `uint8`   | sRGB red component                           |
| `G`     | `uint8`   | sRGB green component                         |
| `B`     | `uint8`   | sRGB blue component                          |
| `L`     | `float64` | CIELAB L* component (precomputed at startup) |
| `A`     | `float64` | CIELAB a* component (precomputed at startup) |
| `BLab`  | `float64` | CIELAB b* component (precomputed at startup) |

### 10.2 Relationships

```
┌─────────────────────────┐
│          Job             │
├─────────────────────────┤
│ ID          string (PK)  │
│ Status      string       │
│ Error       string       │
│ CreatedAt   time.Time    │
│ InputPath   string       │
│ OutputPath  string       │
│ InputName   string       │
└─────────────────────────┘
        │
        │ 1 Job produces 1 output using
        ▼
┌─────────────────────────┐
│     PaletteColor [26]    │
├─────────────────────────┤
│ Name   string            │
│ Hex    string            │
│ R, G, B  uint8           │
│ L, A, BLab  float64      │
└─────────────────────────┘
  (static, loaded once at startup)
```

The `PaletteColor` set is not per-job — it is a global constant initialized at application startup. Every job references the same palette. There is no many-to-many or one-to-many relationship; the palette is a fixed lookup table.

### 10.3 Job Store (sync.Map)

```
Key:   job.ID (string)
Value: *Job (pointer to Job struct)
```

### 10.4 Filesystem Layout (temp dir)

```
/tmp/catppuccinify-XXXXX/
├── {job_id}_original.png   # or .jpg / .webp
├── {job_id}_output.png
├── {job_id2}_original.jpg
├── {job_id2}_output.png
└── ...
```

---

## 11. Non-Functional Requirements

| Requirement      | Specification                                                        |
|------------------|----------------------------------------------------------------------|
| Max upload size  | 10 MB                                                                |
| Input formats    | PNG, JPEG, WebP                                                      |
| Output format    | Always PNG (lossless)                                                |
| Job TTL          | 10 minutes from creation, then auto-deleted                          |
| Polling interval | 2 seconds                                                            |
| Concurrency      | Multiple jobs can process simultaneously (Go HTTP goroutines)        |
| Persistence      | None — all state is ephemeral (in-memory + temp dir)                 |
| Deployment       | Single Go binary + embedded or co-located static files               |
| Target platform  | Any OS with Go support; primary dev on macOS/Linux                   |

---

## 12. Error Handling Summary

| Scenario                   | HTTP Code | User-Facing Message                          |
|----------------------------|-----------|----------------------------------------------|
| No file in upload          | 400       | "No image file provided"                     |
| File too large             | 400       | "File exceeds maximum size of 10 MB"         |
| Invalid image format       | 400       | "Unsupported format. Please upload PNG, JPEG, or WebP" |
| Image decode failure       | 400       | "Could not read image. File may be corrupted"|
| Unknown job ID             | 404       | "Job not found or expired"                   |
| Download before done       | 409       | "Image is still processing"                  |
| Processing failure         | 500       | "Conversion failed. Please try again"        |
| Server error               | 500       | "Internal server error"                      |

---

## 13. Out of Scope (v1)

These are explicitly **not** included in the initial build:

- Additional palettes (Latte, Frappé, Macchiato) — architecture should make adding them easy, but only Mocha is implemented.
- User accounts or authentication.
- Image resize/crop before conversion.
- Dithering toggle (on/off) — always on.
- Batch/multi-image upload.
- WebSocket-based progress (polling is sufficient).
- Docker/containerization (can be added later).
- HTTPS/TLS termination (assumed handled by reverse proxy in production).

---

## 14. Future Considerations (v2+)

When designing the code, keep these in mind for extensibility but do **not** implement them:

- The palette should be defined as a data structure, not scattered constants, so additional Catppuccin flavors (Latte, Frappé, Macchiato) can be added by simply defining new palette arrays.
- The API could accept a `palette` query parameter in the future.
- A dithering strength slider or toggle could be added to the frontend.
- The converter package should be self-contained so it could be reused as a CLI tool or library.
