# Trợ lý AI local trong Docker

DX-OS đóng gói Ollama và model trong Docker Compose. Máy triển khai không cần cài Ollama trực tiếp và không
cần API key của nhà cung cấp LLM. Lần khởi động đầu tiên cần Internet để tải image và model; các lần sau model
được giữ trong named volume `ollama_data`.

## Kiến trúc

```text
Angular /ai-assistant
  -> Go API xác thực JWT và giới hạn câu hỏi
  -> truy xuất đoạn Markdown liên quan trong /app/knowledge
  -> http://ollama:11434 (mạng Docker nội bộ)
  -> qwen3:4b-instruct trong container Ollama
  -> câu trả lời Markdown + danh sách nguồn kiểm chứng
```

Ollama không publish cổng `11434` ra host. Chỉ API trong mạng `dxos_internal` có thể gọi model. Nội dung tài
liệu được coi là dữ liệu không đáng tin cậy; system prompt yêu cầu model bỏ qua chỉ dẫn nằm trong nguồn để giảm
rủi ro prompt injection.

## Khởi động

### Máy có NVIDIA GPU

Docker Desktop trên Windows phải dùng WSL2 và GPU passthrough phải hoạt động:

```powershell
docker run --rm --gpus all ubuntu:24.04 nvidia-smi
docker compose -f docker-compose.yml -f docker-compose.gpu.yml `
  --profile foundation --profile application --profile reporting up -d --build
```

### Máy không có GPU

Bỏ overlay GPU; Ollama tự chạy bằng CPU:

```powershell
docker compose --profile foundation --profile application --profile reporting up -d --build
```

Có thể chỉ khởi động AI bằng script sau. Mặc định script dùng NVIDIA GPU; thêm `-CpuOnly` để dùng CPU:

```powershell
.\scripts\Start-LocalAI.ps1
.\scripts\Start-LocalAI.ps1 -CpuOnly
```

## Model bootstrap và dữ liệu

Service `ollama-init` chạy `ollama pull qwen3:4b-instruct`. Nó chỉ tải các layer còn thiếu, sau đó thoát thành
công. API chỉ khởi động khi bootstrap hoàn tất. Model nằm trong Docker volume, không nằm trong source code:

```powershell
docker compose exec -T ollama ollama list
docker volume inspect dx-os-lab_ollama_data
```

Không dùng `docker compose down -v` nếu muốn giữ database, tài liệu Nextcloud và model đã tải.

Các biến cấu hình:

```dotenv
OLLAMA_IMAGE=ollama/ollama:0.32.15
AI_ENABLED=true
OLLAMA_URL=http://ollama:11434
OLLAMA_CHAT_MODEL=qwen3:4b-instruct
AI_REQUEST_TIMEOUT=90s
```

Muốn mang sang môi trường hoàn toàn offline, cần chuyển cả Docker images và volume model; source repository đơn
lẻ không chứa blob model 2,5 GB.

## Kiểm tra

```powershell
docker compose ps
docker compose exec -T ollama ollama list
docker compose exec -T api wget -qO- http://ollama:11434/api/tags
```

Mở `http://localhost:4200/ai-assistant`. Mọi tài khoản đã xác thực đều có thể dùng trợ lý, nhưng kết quả chỉ
mang tính hỗ trợ và phải kiểm tra nguồn trước khi ra quyết định.

## Giới hạn tài nguyên

- Context cố định ở 4.096 token để phù hợp GPU 4 GB.
- Tối đa 5 đoạn nguồn, tổng khoảng 7.500 ký tự.
- Câu hỏi tối đa 1.000 ký tự.
- Một model và một yêu cầu song song trên cấu hình laptop hiện tại.
- Lần gọi đầu tiên chậm hơn vì Ollama phải nạp model vào VRAM hoặc RAM.
