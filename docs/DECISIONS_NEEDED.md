# Các quyết định còn mở

Go + Angular, modular monolith, PostgreSQL và Keycloak đã được chốt. Các quyết định sau không chặn
việc dựng skeleton nhưng phải chốt trước giai đoạn tương ứng.

## P0 — Trước Sprint 1

1. Số thành viên, vai trò và kinh nghiệm Go/Angular/DevOps?
2. Deadline nghiệm thu và số giờ mỗi tuần?
3. Tên module Go và domain dự kiến?
4. Dùng Git hosting/CI nào?
5. UI chỉ tiếng Việt hay cần song ngữ?

## P1 — Trước Documents/Data

1. File tối đa, loại file cho phép và policy antivirus?
2. KPI chính thức và người xác nhận công thức?
3. Metabase chỉ mở riêng hay embed trong Angular?
4. Có SMTP/notification thật hay chỉ in-app trong MVP?

## P1 — Trước AI

1. LLM và embedding provider?
2. Tài liệu/dữ liệu có được gửi ra dịch vụ bên ngoài?
3. Yêu cầu lưu trữ dữ liệu tại Việt Nam hoặc compliance?
4. RAGFlow chạy máy 32 GB riêng hay chỉ bật theo lịch demo?
5. Ngưỡng chất lượng/refusal cho bộ câu hỏi chuẩn?

## P2 — Trước Production pilot

1. Số user đồng thời, dung lượng và transaction rate?
2. RTO/RPO?
3. Domain, TLS, secret manager và nơi đặt server?
4. Retention cho business data, audit, log và AI trace?
5. Ai là data owner, security owner và on-call owner?

