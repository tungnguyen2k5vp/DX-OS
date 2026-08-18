# Frontend Angular

## 1. Mục tiêu

Angular cung cấp một giao diện thống nhất cho quy trình, dashboard, tài liệu, AI và audit. Frontend
không chứa business rule độc quyền; mọi transition vẫn được backend xác thực.

## 2. Phiên bản chuẩn

- Angular 22, Standalone Components.
- Spartan UI 1.1.x cho accessible primitives và copy-owned helm components.
- Tailwind CSS v4 cho design tokens, layout và responsive.
- Angular CDK cho overlay/accessibility được Spartan sử dụng.
- Signals cho local/feature state và derived state.
- RxJS cho HTTP, event stream và async orchestration.
- Reactive Forms cho biểu mẫu nghiệp vụ.
- Lazy-loaded feature routes.
- TypeScript strict mode.

SSR không cần cho ứng dụng nội bộ. Có thể chạy zoneless sau khi test đầy đủ component/library.

## 3. Cấu trúc

```text
src/app/
├── core/
│   ├── auth/
│   ├── config/
│   ├── http/
│   ├── errors/
│   └── observability/
├── layout/
│   ├── shell/
│   ├── header/
│   ├── sidenav/
│   └── breadcrumb/
├── shared/
│   ├── ui/
│   ├── directives/
│   ├── pipes/
│   └── models/
└── features/
    ├── dashboard/
    ├── procurement/
    ├── approvals/
    ├── documents/
    ├── ai-assistant/
    ├── audit/
    └── administration/
```

Mỗi feature chứa `routes.ts`, `pages/`, `components/`, `data-access/`, `store/` và `models/` khi cần.
Không tạo `SharedModule`; standalone component import trực tiếp dependency. Component Spartan được
copy vào `shared/ui` để nhóm sở hữu và tùy chỉnh source.

## 4. Route map

| Đường dẫn | Màn hình | Vai trò |
|---|---|---|
| `/dashboard` | KPI và việc cần làm | authenticated |
| `/purchase-requests` | danh sách phiếu | employee+ |
| `/purchase-requests/new` | tạo phiếu | employee |
| `/purchase-requests/:id` | chi tiết/timeline | theo scope |
| `/approvals` | hộp việc phê duyệt | manager, finance |
| `/ai` | hỏi đáp và recommendation | authenticated theo policy |
| `/agent-actions` | duyệt/thực thi đề xuất | ai_operator/authorized approver |
| `/audit` | tìm kiếm audit | auditor, dx_admin |
| `/admin` | cấu hình tham chiếu | dx_admin |

Route guard kiểm authentication/role để điều hướng; backend vẫn là policy enforcement point cuối.

## 5. State management

Signals dùng cho:

- filter và paging hiện tại;
- loading/error;
- selected request;
- derived actions có thể hiển thị;
- unread count và UI preference.

RxJS dùng cho:

- `HttpClient`;
- debounce search;
- cancel request cũ bằng `switchMap`;
- polling trạng thái job khi cần;
- orchestration nhiều async source.

Không đưa toàn ứng dụng vào một global store ngay từ đầu. Mỗi feature có store/service riêng; auth và
notification là global state tối thiểu.

## 6. HTTP client

Interceptor chịu trách nhiệm:

- gắn access token;
- gửi `X-Correlation-ID`;
- bắt 401 để bắt đầu login/renew phù hợp;
- map Problem Details;
- không retry tự động POST transition;
- không log body nhạy cảm.

Typed client được sinh hoặc đồng bộ từ OpenAPI. DTO transport không dùng trực tiếp làm form model nếu
cần trạng thái UI riêng.

## 7. Form tạo phiếu

Reactive Form gồm:

- title, reason, cost center, currency;
- `FormArray` items: description, quantity, unit price;
- total là computed/read-only;
- attachment metadata;
- client validation để UX tốt;
- server validation là nguồn quyết định cuối.

Draft có thể lưu thủ công. Khi submit, UI hiển thị validation server theo field và warning về
attachment/ngân sách.

## 8. Trang chi tiết

Bố cục desktop:

```text
┌──────────────── Header / Breadcrumb ────────────────┐
│ Mã phiếu | trạng thái | tổng tiền | hành động       │
├───────────────────────┬─────────────────────────────┤
│ Thông tin và items    │ Timeline / người xử lý      │
│ Ngân sách             │ Comments                    │
│ Attachments           │ AI recommendation           │
└───────────────────────┴─────────────────────────────┘
```

Mobile/tablet chuyển thành tab/accordion; không cố giữ bảng rộng.

## 9. Action và concurrency

- Backend trả `version` cùng request.
- Transition gửi `expectedVersion` và `Idempotency-Key`.
- Nếu 409, UI tải lại dữ liệu và thông báo hồ sơ đã được người khác cập nhật.
- Nút action dựa trên `allowedActions` backend trả về, không tự suy luận toàn bộ rule.
- Double-click bị khóa trong lúc request đang chạy.

## 10. Accessibility và UX

- Keyboard navigation và focus management cho dialog.
- Label/error rõ cho form control.
- Color không phải tín hiệu duy nhất của trạng thái.
- Loading skeleton cho trang; spinner nhỏ cho action.
- Empty/error state có hành động khôi phục.
- Ngôn ngữ mặc định tiếng Việt; chuỗi UI tách để sẵn sàng i18n.

## 11. Quy tắc component system

- Spartan UI là component system duy nhất; không trộn Angular Material.
- Ưu tiên component đã có trước khi tự viết primitive mới.
- Chỉ sửa helm component cho yêu cầu dùng chung; business component nằm trong feature.
- Màu, radius, spacing và typography lấy từ CSS variables/theme token.
- Không hard-code màu trạng thái ở nhiều component.
- Mỗi component copy vào source phải có accessibility test phù hợp.

Chi tiết tại [UI Design System](UI_DESIGN_SYSTEM.md).

## 12. Test frontend

- Unit test component/store/pure mapper.
- Form validation test.
- HTTP test với mock backend.
- Guard và role visibility test.
- E2E cho create -> submit -> approve -> finance approve.
- Accessibility smoke test cho login callback, form và approval dialog.
