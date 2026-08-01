# ADR-005: State machine trong Go thay workflow engine

## Status

Accepted cho MVP

## Context

MVP chỉ có một quy trình mua sắm với số trạng thái/transition hữu hạn. Thêm Flowable hoặc engine
khác làm tăng dịch vụ, mô hình vận hành và tích hợp UI.

## Decision

Mô hình hóa transition tường minh trong Go. Mỗi transition kiểm role, scope, ownership, trạng thái,
precondition, optimistic version và idempotency; sau đó ghi event/audit/outbox trong một transaction.

## Trade-offs

- Đơn giản, dễ test và debug.
- Thay đổi workflow cần code/deploy.
- Không có BPMN designer hoặc process analytics sẵn.

## Revisit trigger

- Có từ ba quy trình phức tạp trở lên.
- Người nghiệp vụ cần tự chỉnh workflow.
- Có timer/parallel/subprocess/escalation vượt khả năng state machine đơn giản.

