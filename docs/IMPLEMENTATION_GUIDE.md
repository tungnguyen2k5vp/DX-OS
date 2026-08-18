---
title: Hướng dẫn triển khai Go + Angular
sidebar_position: 5
---

# DX-OS — Hướng dẫn triển khai Go + Angular

## 1. Mục tiêu

DX-OS là nền tảng vận hành doanh nghiệp số, không phải hệ điều hành máy tính. MVP phải chứng minh
một quy trình đề nghị mua sắm nội bộ xuyên suốt bốn lớp:

- **Human:** Keycloak quản lý danh tính, role và đăng nhập.
- **Process:** Go API thực thi state machine, rule và phân công.
- **Data:** PostgreSQL lưu dữ liệu nghiệp vụ, event, audit và dữ liệu báo cáo.
- **Intelligence:** RAGFlow trả lời có nguồn, phân tích và đề xuất; mọi hành động nhạy cảm cần người
  phê duyệt.

```text
Đăng nhập -> tạo phiếu -> gửi -> trưởng bộ phận duyệt
-> tài chính duyệt -> dashboard cập nhật
-> AI phân tích -> người có quyền xác nhận -> tool thực thi -> audit
```

## 2. Kiến trúc đã chọn

Ứng dụng do nhóm tự xây gồm:

- Angular SPA cho giao diện.
- Go modular monolith cho REST API và nghiệp vụ.
- PostgreSQL cho dữ liệu.
- Keycloak cho OIDC/RBAC.

Các nền tảng dùng lại:

- Nextcloud cho file và version tài liệu.
- Metabase cho dashboard.
- RAGFlow cho RAG và Agent.
- Nginx hoặc reverse proxy tương đương cho TLS, static Angular và routing.

NocoBase không còn thuộc kiến trúc đích. Quyết định này loại bỏ phụ thuộc vào plugin OIDC/Approval
thương mại và giúp đồ án thể hiện rõ năng lực lập trình Go + Angular.

## 3. Baseline công nghệ

| Thành phần   | Baseline phát triển                    |
| ------------ | -------------------------------------- |
| Go           | 1.26.6                                 |
| Angular      | 22.0.x                                 |
| Angular CLI  | 22.0.x                                 |
| Spartan UI   | 1.1.2                                  |
| Angular CDK  | 22.0.6                                 |
| Tailwind CSS | 4.3.3                                  |
| Node.js      | 24.15+                                 |
| TypeScript   | phiên bản 6.0.x tương thích Angular 22 |
| PostgreSQL   | 18.4                                   |
| Keycloak     | 26.7.0                                 |
| Nextcloud    | 34.0.2                                 |
| Metabase OSS | 0.63.1                                 |
| RAGFlow      | 0.26.4                                 |

Version chính xác được khóa trong `go.mod`, lockfile frontend và `.env.example`. Không dùng tag
`latest` cho bản demo/UAT hoặc production pilot.

## 4. Nguyên tắc thiết kế

1. **Modular monolith trước:** một Go binary, module nghiệp vụ tách rõ, một database nghiệp vụ.
2. **Không microservices sớm:** chỉ tách service khi có nhu cầu scale/vòng đời độc lập đã đo được.
3. **REST synchronous cho MVP:** webhook/outbox cho tích hợp; chưa cần Kafka/RabbitMQ.
4. **State machine tường minh:** transition hợp lệ nằm trong Go, không cập nhật `status` trực tiếp.
5. **Backend quyết định quyền:** Angular chỉ ẩn/hiện UI; Go API vẫn kiểm role, ownership và trạng thái.
6. **AI không được tin mặc định:** chỉ đề xuất, tool nằm trong allowlist, input được validate.
7. **Quan sát được:** mọi request có correlation ID, transition có event và thao tác quan trọng có audit.

## 5. Phạm vi MVP

### Có trong MVP

- OIDC login và logout.
- Role: employee, department manager, finance, auditor, dx admin, ai operator.
- Tạo và chỉnh sửa phiếu ở trạng thái draft/changes requested.
- Hạng mục mua sắm, ngân sách và file báo giá.
- Submit, approve, reject, request changes và cancel.
- Rule người tạo không tự duyệt.
- Timeline, comment, notification tối thiểu.
- Dashboard lead time, tỷ lệ trả lại, SLA và ngân sách.
- RAG trả lời có nguồn.
- AI recommendation, human approval, tool execution và audit.
- Seed/reset dữ liệu demo, backup/restore và test report.

### Không thuộc MVP

- Microservices, Kubernetes hoặc multi-region.
- Kafka, CQRS hoặc event sourcing.
- LLM local trên máy 16 GB.
- Flowable/Airbyte/OpenMetadata/Mattermost.
- Mobile app native.
- Workflow designer kéo-thả cho người dùng.

## 6. Các giai đoạn

| Giai đoạn      | Kết quả bắt buộc                                   |
| -------------- | -------------------------------------------------- |
| 0. Chuẩn bị    | tài liệu, ADR, repo, baseline, backlog             |
| 1. Foundation  | PostgreSQL + Keycloak healthy; role có trong token |
| 2. Skeleton    | Go `/health`, Angular shell, OIDC login, CI build  |
| 3. Process MVP | tạo/gửi/duyệt/từ chối/yêu cầu sửa đúng quyền       |
| 4. Documents   | file Nextcloud gắn đúng hồ sơ                      |
| 5. Data/BI     | curated data và dashboard khớp dữ liệu gốc         |
| 6. RAG         | bộ câu hỏi chuẩn, trả lời có citation              |
| 7. Agent       | recommendation -> approval -> execute -> audit     |
| 8. Hardening   | security, observability, restore và demo ổn định   |

Chi tiết công việc nằm tại [BACKLOG.md](BACKLOG.md).

## 7. Definition of Done

Một story chỉ hoàn thành khi:

- acceptance criteria chạy được;
- unit/integration test phù hợp đã pass;
- API/OpenAPI và migration được cập nhật;
- không có secret trong source;
- log không chứa token/PII không cần thiết;
- quyền backend và negative test đã có;
- tài liệu/runbook được cập nhật;
- có rollback hoặc restore plan;
- build lại được trên môi trường sạch.

## 8. Quy trình phát triển

```text
Issue có acceptance criteria
-> cập nhật contract/schema nếu cần
-> migration + backend
-> frontend
-> unit/integration/E2E
-> security review
-> cập nhật docs và bằng chứng
-> merge
```

Mỗi thay đổi phá vỡ contract, authentication, data ownership hoặc deployment topology cần ADR mới.
