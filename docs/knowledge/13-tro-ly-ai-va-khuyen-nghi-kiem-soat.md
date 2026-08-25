# Trợ lý AI nội bộ và Trung tâm khuyến nghị

## Trợ lý AI nội bộ

Trang `/ai-assistant` dành cho mọi người dùng đã đăng nhập. Go API tìm các đoạn Markdown liên quan trong thư mục `docs`, ghép tối đa các nguồn phù hợp vào câu hỏi rồi gọi model Ollama local. Câu trả lời kèm danh sách nguồn để người dùng kiểm chứng.

Kho tri thức chỉ đọc tệp `.md` tối đa 2 MB. Các thư mục `generated`, `superpowers` và `drift-audit` bị bỏ qua. Tài liệu được nạp một lần khi API khởi động; sau khi thêm hoặc sửa tài liệu phải khởi động lại container API.

Trợ lý AI không tự gọi API nghiệp vụ và không tự thay đổi phiếu, ngân sách, đơn hàng hoặc thanh toán.

## Trung tâm khuyến nghị

Trang `/ai-center` dành cho Điều phối AI, Quản trị, Tài chính và Kiểm toán. Các khuyến nghị hiện được tạo bằng quy tắc SQL có thể giải thích, không phải quyết định bí mật của mô hình ngôn ngữ.

Các nhóm kiểm soát gồm: quá hạn xử lý; phiếu từ 50 triệu VND; nhà cung cấp rủi ro/tuân thủ không đạt; phiếu có khả năng trùng; chia nhỏ nhu cầu trong 7 ngày; đơn giá cao hơn lịch sử; hóa đơn quá hạn; thông tin nhà cung cấp thay đổi sau khi đặt hàng; tài khoản có vai trò xung đột đã thao tác.

`ai_operator` và `dx_admin` có thể tạo đợt quét và quyết định khuyến nghị, trừ tài khoản đồng thời mang vai trò Kiểm toán. Tài chính và Kiểm toán xem được nhưng không vận hành hàng đợi. Quyết định Chấp nhận, Bác bỏ hoặc Bỏ qua cần lý do và được ghi audit; nó không tự thực hiện hành động nghiệp vụ.

## Nguồn mã xác minh

`backend/internal/aiassistant/service.go`, `backend/internal/procurement/ai_recommendations.go`, `frontend/src/app/features/ai`.
