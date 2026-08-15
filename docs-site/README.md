# DX-OS Documentation Portal

Website tài liệu dùng Docusaurus 3. Nội dung được đọc trực tiếp từ thư mục ../docs để tránh tạo hai
nguồn tài liệu khác nhau.

Website production: https://tungnguyen2k5vp.github.io/DX-OS/

## Chạy local

```powershell
npm ci
npm start
```

Mở http://localhost:3000 khi chạy dev server trực tiếp. Có thể đổi port bằng:

```powershell
npm start -- --port 4300
```

## Kiểm tra production build

```powershell
npm run typecheck
npm run build
npm run serve -- --port 4300
```

Build thất bại nếu có link nội bộ hoặc Markdown link bị hỏng.

## Docker

Từ repository root:

```powershell
docker compose --profile documentation up -d --build docs
```

Mở http://localhost:4300.

Smoke test từ repository root:

```powershell
.\scripts\Test-Documentation.ps1
```

## Triển khai GitHub Pages

Workflow `.github/workflows/deploy-docs.yml` tự động build và deploy khi nội dung trong `docs/` hoặc
`docs-site/` được push lên nhánh `main`. Trong lần thiết lập đầu tiên, vào **Settings → Pages → Build
and deployment → Source**, chọn **GitHub Actions**. Sau đó có thể chạy workflow thủ công tại tab
**Actions → Deploy DX-OS Docs to GitHub Pages** hoặc push một thay đổi tài liệu mới.

Build production tại máy local (PowerShell):

```powershell
$env:DOCS_SITE_URL = 'https://tungnguyen2k5vp.github.io'
$env:DOCS_BASE_URL = '/DX-OS/'
$env:DOCS_APP_URL = 'https://github.com/tungnguyen2k5vp/DX-OS'
$env:DOCS_APP_LABEL = 'Mã nguồn'
npm run build
```

Khi không đặt các biến trên, website tiếp tục chạy local với `baseUrl=/` và liên kết ứng dụng
`http://localhost:4200`.

## Quy ước

- docusaurus.config.ts là nguồn cấu hình duy nhất.
- Sidebar tự sinh từ cấu trúc ../docs.
- Dùng front matter và _category_.json để điều khiển thứ tự.
- Ưu tiên custom CSS và CSS variables; chỉ swizzle bằng --wrap khi thật sự cần.
- Không dùng --eject nếu chưa có lý do kỹ thuật được ghi nhận.
- Tài liệu Markdown thuần dùng .md; chỉ dùng .mdx khi cần component React.
