# ADR-002: Gate SSO và Approval của NocoBase

## Trạng thái

Superseded by [ADR-004](ADR-004-go-angular-application.md)

## Bối cảnh

MVP yêu cầu Keycloak SSO và workflow phê duyệt. Tại ngày 2026-07-29, tài liệu NocoBase ghi
`Auth: OIDC` và `Workflow: Approval` thuộc Professional Edition. Community Edition vẫn có
authentication nền tảng, workflow cơ bản, custom action và khả năng mở rộng plugin.

## Options considered

| Phương án | Ưu điểm | Nhược điểm | Khi phù hợp |
|---|---|---|---|
| Mua NocoBase Professional | nhanh nhất, được hỗ trợ | chi phí/license, phụ thuộc edition | có ngân sách và cần demo chắc chắn |
| Tự viết plugin OIDC + state-machine approval | giữ mục tiêu kỹ thuật/open-core | tăng effort và rủi ro 2-4 tuần | đồ án cần chứng minh năng lực tích hợp |
| Dùng login nội bộ ở MVP, SSO sau | giảm rủi ro đầu kỳ | không đạt tiêu chí nghiệm thu SSO | chỉ làm spike ngắn |
| Chuyển workflow sang Flowable/app custom | BPMN/open-source rõ hơn | thêm dịch vụ và UI tích hợp | workflow là trọng tâm hoặc không mua license |

## Kết quả

Nhóm đã chọn xây ứng dụng bằng Go + Angular. NocoBase không còn trong kiến trúc đích; Keycloak tích
hợp trực tiếp với Angular/Go và approval được mô hình hóa bằng state machine trong Go.
