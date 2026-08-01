# UI Design System — Spartan UI

## 1. Quyết định

DX-OS dùng Angular 22, Spartan UI 1.1.x, Tailwind CSS v4, Angular CDK, Signals và Reactive Forms.
Không dùng package shadcn/ui React trực tiếp và không trộn Angular Material.

Spartan giữ triết lý open-code của shadcn:

- behavior/accessibility primitive được cài từ `@spartan-ng/brain`;
- helm component/style được copy vào source;
- nhóm sở hữu và tùy chỉnh code giao diện;
- theme được điều khiển bằng CSS variables.

## 2. Cấu trúc

```text
frontend/src/
├── app/
│   ├── shared/
│   │   └── ui/               # Spartan helm components
│   ├── layout/               # shell, header, sidebar
│   └── features/             # business components
└── styles.css                # Tailwind layers + DX-OS tokens
```

`shared/ui` chỉ chứa primitive/generic component. Ví dụ `PurchaseStatusBadge` thuộc feature
`purchase-requests`, không đặt trong `shared/ui`.

## 3. Cài đặt

```bash
npm install --save-dev @spartan-ng/cli@1.1.2
ng g @spartan-ng/cli:init
ng g @spartan-ng/cli:ui
```

CLI sẽ cài `@spartan-ng/brain`, Angular CDK/dependency cần thiết, copy helm component vào source và
tạo/cập nhật `components.json`.

Sau khi sinh component, phải review diff như source nội bộ; không mặc định chấp nhận toàn bộ code
được generate.

## 4. Tailwind layers

`styles.css` giữ thứ tự:

```css
@layer theme, base, components, utilities;
@import "tailwindcss/theme.css" layer(theme);
@import "tailwindcss/preflight.css" layer(base);
@import "tailwindcss/utilities.css";
@import "@spartan-ng/brain/hlm-tailwind-preset.css";
```

Không dùng `!important` toàn cục. Nếu cần variant mới, thêm variant trong helm component hoặc
business wrapper có phạm vi rõ.

## 5. Theme DX-OS

Theme dùng CSS variables tương thích Spartan:

```text
--background / --foreground
--card / --card-foreground
--primary / --primary-foreground
--secondary / --secondary-foreground
--muted / --muted-foreground
--accent / --accent-foreground
--destructive
--border / --input / --ring
--sidebar*
--radius
```

Quy ước:

- Light mode là mặc định; dark mode được hỗ trợ nhưng không chặn MVP.
- Primary dùng xanh lam hoặc teal tạo cảm giác tin cậy.
- Destructive chỉ dùng cho reject/delete/cancel nghiêm trọng.
- Success, warning, info và process-state có token riêng.
- Không hard-code màu hex trong feature component.

## 6. Typography và spacing

- Font sans dễ đọc, hỗ trợ đầy đủ tiếng Việt.
- Body mặc định 14–16 px tùy mật độ dashboard.
- Heading có hierarchy đúng semantic.
- Form desktop có max-width hợp lý.
- Spacing dùng Tailwind scale; hạn chế arbitrary value.

## 7. Component MVP

| Nhu cầu | Spartan component |
|---|---|
| App shell | Sidebar, Sheet, Breadcrumb |
| KPI | Card, Badge, Skeleton |
| Danh sách | Table/Data Table, Pagination, Dropdown Menu |
| Form | Field, Input, Textarea, Select, Combobox, Checkbox |
| Action | Button, Alert Dialog, Dialog |
| Timeline | Card, Separator, Tooltip |
| Chi tiết | Tabs, Accordion/Collapsible |
| Notification | Sonner/Toast, Alert |
| Loading | Skeleton, Spinner, Progress |

Chỉ add component khi có use case; không generate toàn bộ thư viện ngay từ đầu.

## 8. Business components

- `PurchaseStatusBadge`
- `MoneyDisplay`
- `RequestItemsEditor`
- `RequestTimeline`
- `AllowedActions`
- `ApprovalDecisionDialog`
- `AttachmentUploader`
- `BudgetUsageCard`
- `RecommendationCard`
- `CitationList`
- `AuditEventTable`

Business component compose Spartan primitives và chứa presentation logic. API call nằm ở page/store/
data-access, không nằm trực tiếp trong primitive.

## 9. Trạng thái phiếu

| Status | Nhãn UI | Intent |
|---|---|---|
| `DRAFT` | Bản nháp | neutral |
| `SUBMITTED` | Chờ trưởng bộ phận | warning |
| `MANAGER_APPROVED` | Chờ tài chính | info |
| `CHANGES_REQUESTED` | Cần bổ sung | warning |
| `APPROVED` | Đã duyệt | success |
| `REJECTED` | Bị từ chối | destructive |
| `CANCELLED` | Đã hủy | muted |

Màu không phải tín hiệu duy nhất; luôn có text/icon phù hợp.

## 10. Layout

Desktop:

- sidebar trái có thể thu gọn;
- header có breadcrumb, notification và user menu;
- content width linh hoạt;
- primary action nhất quán ở page header.

Tablet/mobile:

- sidebar chuyển thành Sheet;
- bảng quan trọng dùng responsive columns hoặc card list;
- form item chuyển stacked layout;
- action bar không che nội dung.

## 11. Form

- Reactive Forms là nguồn trạng thái.
- Label liên kết control.
- Required/optional thể hiện rõ.
- Error server map về field hoặc form summary.
- Destructive/approval action cần confirmation.
- Submit khóa khi đang xử lý, nhưng backend idempotency vẫn bắt buộc.

## 12. Accessibility

- Keyboard navigation cho menu, dialog và table action.
- Focus trap/focus return đúng.
- Contrast đạt WCAG AA.
- Icon button có accessible name.
- Dialog có title/description.
- Toast không phải nơi duy nhất chứa lỗi quan trọng.
- Screen reader đọc được status và validation.

## 13. Tùy chỉnh và nâng cấp

Thứ tự ưu tiên:

1. dùng component hiện có;
2. dùng variant/theme token;
3. compose business component;
4. chỉ sửa helm primitive nếu thay đổi có ích toàn hệ thống.

Khi nâng Spartan, không overwrite mù helm code đã chỉnh. Review diff và chạy visual/accessibility
regression.

## 14. Test

- Unit test behavior/variant quan trọng.
- Form keyboard/focus test.
- Accessibility smoke tự động và kiểm tra thủ công.
- Visual regression cho shell, form, table, dialog và dark mode.
- Responsive test tại mobile/tablet/desktop breakpoints.

