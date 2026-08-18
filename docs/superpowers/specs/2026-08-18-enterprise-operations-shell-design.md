# Đặc tả thiết kế DX-OS Enterprise Operations Shell

**Ngày:** 2026-08-18
**Trạng thái:** Chờ duyệt đặc tả
**Phương án được chọn:** A — Enterprise Operations Shell
**Phạm vi:** Thiết kế lại toàn bộ frontend DX-OS; giữ nguyên backend, API, workflow và RBAC hiện có.

## 1. Mục tiêu

Thiết kế lại DX-OS thành một sản phẩm vận hành doanh nghiệp rõ ràng, đáng tin cậy và có mật độ thông tin phù hợp với công việc hằng ngày. Giao diện mới phải giúp mỗi vai trò nhận ra ngay việc cần làm, giảm cảm giác dashboard mẫu do AI tạo và giải quyết triệt để vấn đề menu ngang quá dài.

Thành công được đo bằng các kết quả sau:

- Người dùng xác định được việc ưu tiên trong vòng vài giây sau khi đăng nhập.
- Điều hướng không bị tràn, xuống hai dòng hoặc phụ thuộc vào cuộn ngang.
- Dashboard của từng vai trò phản ánh đúng nhiệm vụ nghiệp vụ thay vì dùng chung một mẫu KPI.
- Các màn hình danh sách, chi tiết, form và hàng đợi dùng cùng một ngôn ngữ thiết kế.
- Không thay đổi hợp đồng API, trạng thái nghiệp vụ, route hiện có hoặc quyền truy cập.
- Các luồng quan trọng sử dụng được bằng bàn phím và thích ứng tốt từ mobile đến desktop.

## 2. Hiện trạng và nguyên nhân gốc

Khảo sát frontend hiện tại ghi nhận 17 trang chính, khoảng 93 card và 17 bảng. Phần lớn màn hình lặp lại cấu trúc tiêu đề lớn, cụm KPI card và bảng có chiều rộng tối thiểu lớn. App shell nhân đôi menu cho desktop/mobile và hiển thị tất cả module được cấp quyền trên một thanh ngang.

Nguyên nhân gốc cần xử lý:

1. Kiến trúc thông tin phẳng: module không được nhóm theo ngữ cảnh công việc.
2. Một mẫu dashboard được dùng cho nhiều persona, chỉ thay nhãn và số liệu.
3. Card được dùng như đơn vị bố cục mặc định, kể cả khi section phẳng hoặc bảng sẽ phù hợp hơn.
4. Phân cấp typography và khoảng trắng thiên về landing page hơn là phần mềm tác nghiệp.
5. Bảng responsive dựa nhiều vào `min-width` và cuộn ngang.
6. Nội dung kỹ thuật như trạng thái nền tảng và danh tính chiếm vị trí nổi bật trên dashboard nghiệp vụ.

## 3. Persona và công việc cốt lõi

### Employee

Mục tiêu chính: tạo/gửi phiếu đúng, sửa phiếu được trả lại, theo dõi trạng thái và xác nhận nhận hàng.

Dashboard ưu tiên: việc cần xử lý, bản nháp, yêu cầu chỉnh sửa, giao nhận đang chờ và phiếu gần đây. CTA chính là **Tạo phiếu mua sắm**.

### Department Manager

Mục tiêu chính: xử lý hàng đợi phê duyệt theo SLA, xem đủ bối cảnh ngân sách/rủi ro và theo dõi hoạt động của phòng ban.

Dashboard ưu tiên: phiếu chờ quyết định, gần quá SLA, giá trị chờ duyệt và ngoại lệ phòng ban. CTA chính là **Mở hàng đợi phê duyệt**.

### Finance

Mục tiêu chính: kiểm soát ngân sách, hoàn tất bước tài chính, đặt hàng, đối soát hóa đơn và theo dõi thanh toán.

Dashboard ưu tiên: ngoại lệ ngân sách, phiếu chờ tài chính, đơn hàng bị nghẽn, hóa đơn sai lệch/quá hạn và khoản sẵn sàng thanh toán.

### Auditor

Mục tiêu chính: đọc bằng chứng, phát hiện sai lệch, theo dõi khắc phục và xuất hồ sơ kiểm toán mà không làm thay đổi nghiệp vụ.

Dashboard ưu tiên: phát hiện mở, hồ sơ thiếu bằng chứng, khắc phục quá hạn và phạm vi kiểm toán. Tất cả khu vực phải thể hiện rõ chế độ **Chỉ xem** khi thích hợp.

### DX Admin

Mục tiêu chính: quản lý người dùng, phòng ban, chính sách và sức khỏe cấu hình hệ thống.

Dashboard ưu tiên: tài khoản/cấu hình cần chú ý, thay đổi chính sách gần đây và lỗi tích hợp. Không trộn KPI mua sắm vào khu vực quản trị nếu không phục vụ quyết định quản trị.

### AI Operator

Mục tiêu chính: xem xét khuyến nghị, đánh giá độ tin cậy/tác động, chấp nhận hoặc bác bỏ có lý do và bảo đảm truy vết.

Dashboard ưu tiên: khuyến nghị cần đánh giá, mức rủi ro, độ tin cậy, nguồn giải thích và lịch sử quyết định.

## 4. Kiến trúc thông tin

Điều hướng được tổ chức theo nhóm nghiệp vụ. Chỉ hiển thị nhóm khi người dùng có ít nhất một mục con được phép truy cập.

| Nhóm | Mục điều hướng | Route hiện tại |
| --- | --- | --- |
| Không gian làm việc | Tổng quan | `/dashboard` |
| Công việc | Việc của tôi, Phê duyệt, Thông báo | `/work-center`, `/approvals`, `/notifications` |
| Mua sắm | Phiếu mua sắm, Đặt hàng & giao nhận, Nhà cung cấp | `/purchase-requests`, `/operations`, `/suppliers` |
| Tài chính | Ngân sách, Hóa đơn & thanh toán, Báo cáo | `/budgets`, `/invoices`, `/reports` |
| Kiểm soát | Kiểm toán, Chính sách | `/audit`, `/policies` |
| Trí tuệ hỗ trợ | Khuyến nghị | `/ai-center` |
| Hệ thống | Quản trị | `/admin` |

`/employee-guide` được đặt trong menu trợ giúp theo ngữ cảnh của Employee thay vì chiếm một mục cấp cao. Route và guard hiện tại vẫn giữ nguyên.

## 5. App shell

### Desktop từ 1280px

- Sidebar trái rộng khoảng 256px, có thể thu gọn còn 72px.
- Logo, tên sản phẩm và tên không gian ở đầu sidebar.
- Navigation chia nhóm; nhóm và mục đang chọn có trạng thái rõ ràng.
- Cuối sidebar có trợ giúp, trạng thái thu gọn và phiên bản sản phẩm nếu cần.
- Utility header nằm trên nội dung, chứa breadcrumb, thông báo và account menu.
- Nội dung tối đa khoảng 1440px; màn hình bảng có thể dùng chiều rộng linh hoạt hơn.

### Tablet

- Sidebar mặc định thu gọn.
- Panel phụ và bộ lọc nâng cao chuyển thành drawer/sheet.
- Không làm mất thao tác chính khi không đủ chiều rộng.

### Mobile

- Top app bar ngắn với logo, tiêu đề trang, thông báo và nút mở menu.
- Menu mở trong modal sheet có focus trap và nút đóng rõ ràng.
- Nội dung một cột; CTA chính có thể sticky ở đáy khi an toàn.
- Không dùng thanh menu ngang cuộn.

### Trạng thái và lưu trữ

- Trạng thái thu gọn sidebar được lưu cục bộ theo trình duyệt.
- Việc ẩn/hiện module luôn được tính từ quyền hiện tại, không dựa vào trạng thái UI đã lưu.
- Khi đổi route, focus được đưa đến heading chính của trang hoặc vùng nội dung phù hợp.

## 6. Ngôn ngữ thiết kế

### Màu sắc

- Nền trung tính sáng, không dùng gradient trang trí ở màn hình tác nghiệp.
- Teal trầm là màu nhận diện và CTA chính.
- Xanh lá, hổ phách, đỏ và xanh dương chỉ dùng theo ngữ nghĩa trạng thái.
- Mọi trạng thái phải có chữ hoặc biểu tượng; màu không phải tín hiệu duy nhất.

### Typography

- Font sans dễ đọc, ưu tiên Inter/system font hiện tại.
- Page title: 24–28px; section title: 16–18px; body/data: 14–16px.
- Số liệu tài chính dùng tabular numerals và căn phải.
- Không dùng heading 36–40px trên màn hình thao tác thông thường.

### Hình khối và chiều sâu

- Radius theo ba mức 6px, 10px và 14px.
- Card chỉ dùng để nhóm một đơn vị thông tin độc lập.
- Section nội dung thông thường dùng border/divider phẳng.
- Shadow chỉ dùng cho popover, sheet, dialog và vùng sticky cần tách lớp.

### Chuyển động

- Chuyển động 120–200ms, phục vụ phản hồi trạng thái.
- Không dùng hiệu ứng trang trí liên tục.
- Tôn trọng `prefers-reduced-motion`.

## 7. Bộ component mục tiêu

Các component dùng Angular 22, Tailwind CSS 4 và Spartan hiện có; không đưa React hoặc package shadcn/ui vào dự án.

- `AppShell`, `Sidebar`, `SidebarGroup`, `SidebarItem`
- `UtilityHeader`, `Breadcrumb`, `AccountMenu`, `NotificationButton`
- `PageHeader`, `RoleContext`, `PrimaryAction`
- `MetricStrip`, `WorkQueue`, `FilterBar`, `DataTable`
- `StatusChip`, `MoneyCell`, `DeadlineIndicator`, `ActionMenu`
- `DetailSection`, `DecisionPanel`, `ActivityTimeline`, `CommentThread`
- `FormSection`, `FieldMessage`, `StickySummary`, `StepIndicator`
- `EmptyState`, `InlineAlert`, `SkeletonRows`, `ErrorState`
- `Dialog`, `Drawer/Sheet`, `ConfirmAction`

Ưu tiên trích xuất component từ các mẫu lặp đang tồn tại; không tạo abstraction tổng quát trước khi có ít nhất hai nơi dùng thực tế.

## 8. Mẫu màn hình

### Dashboard theo vai trò

Mỗi dashboard gồm:

1. Context header ngắn: lời chào, vai trò/phòng ban và một CTA chính.
2. Metric strip tối đa 3–4 chỉ số phục vụ quyết định.
3. Hàng đợi công việc chính, hiển thị trước phần phân tích.
4. Ngoại lệ/cảnh báo liên quan vai trò.
5. Dữ liệu gần đây hoặc tóm tắt thứ cấp.

Không hiển thị “Trạng thái nền tảng” hay thông tin API cho Employee/Manager/Finance/Auditor trừ khi có lỗi thực sự ảnh hưởng công việc.

### Danh sách và hàng đợi

- Page header gọn, CTA ở bên phải.
- Filter bar thống nhất: tìm kiếm, trạng thái, khoảng thời gian và bộ lọc theo vai trò.
- Bảng có caption ẩn trực quan, header sticky, `scope="col"`, số tiền căn phải.
- Hành động hàng dùng một primary inline action hoặc action menu; tránh nhiều nút ngang nhau.
- Mobile ưu tiên cột mã phiếu, nội dung, trạng thái và việc cần làm; trường phụ mở trong row detail.

### Chi tiết phiếu

- Cột chính chiếm khoảng hai phần ba: lý do, hàng hóa, tài liệu, trao đổi.
- Cột phụ sticky: trạng thái, ngân sách, thông tin người yêu cầu và hành động theo vai trò.
- Lịch sử xử lý có thể thu gọn nhưng luôn truy cập được.
- Với Auditor, panel hành động chuyển thành panel phạm vi/chứng cứ và gắn nhãn chỉ xem.

### Tạo/sửa phiếu

- Chia nội dung thành các bước hoặc section có thứ tự rõ ràng.
- Sticky summary hiển thị số dòng, tổng tạm tính và lỗi cần sửa.
- “Lưu bản nháp” và “Gửi phê duyệt” có phân cấp rõ; không đánh đồng hai hành động.
- Lỗi hiển thị cạnh trường và có error summary khi gửi form thất bại.

## 9. Trạng thái giao diện

Mọi trang dữ liệu phải có đủ:

- Loading bằng skeleton phù hợp cấu trúc thật.
- Empty state giải thích nguyên nhân và gợi ý hành động nếu có.
- Error state bằng tiếng Việt, không hiển thị stack trace hoặc chi tiết kỹ thuật không cần thiết.
- Success feedback ngắn, có thể được công nghệ hỗ trợ đọc qua live region.
- Disabled state có lý do khi người dùng cần hiểu vì sao chưa thể thao tác.
- Optimistic UI chỉ dùng khi có thể hoàn tác an toàn; quyết định nghiệp vụ quan trọng chờ phản hồi server.

## 10. Accessibility

Mục tiêu triển khai là WCAG 2.2 AA, nhưng chỉ xác nhận sau kiểm thử tự động và thủ công.

- Thêm skip link đến nội dung chính.
- Duy trì thứ tự heading và landmark hợp lý.
- Bảng có caption, scope và tên truy cập rõ ràng.
- Focus indicator rõ; route, dialog, drawer và menu quản lý focus đúng.
- Toàn bộ thao tác chính dùng được bằng bàn phím.
- Dialog hỗ trợ Escape, focus trap và trả focus về trigger.
- Contrast văn bản/trạng thái được đo, không ước lượng bằng mắt.
- Target tương tác mobile đủ lớn và có khoảng cách an toàn.
- Kiểm tra với reduced motion, zoom 200%, reflow 320 CSS px và screen reader.

## 11. Tính tương thích và ranh giới

### Bắt buộc giữ nguyên

- Angular 22, Tailwind CSS 4, Spartan và cấu trúc standalone component hiện tại.
- URL/route công khai hiện có.
- Keycloak, role guard và logic phân quyền backend.
- API contract, trạng thái phiếu và quy trình nghiệp vụ.
- Dữ liệu và hành động mà từng vai trò đang được phép sử dụng.

### Không thuộc phạm vi

- Thiết kế lại backend hoặc database.
- Thay đổi chính sách phê duyệt/ngân sách.
- Thêm framework UI React hoặc sao chép package shadcn/ui.
- Thay đổi nhận diện thương hiệu ngoài giao diện sản phẩm.
- Commit, push hoặc merge vào `main` nếu chưa được yêu cầu riêng.

## 12. Chiến lược triển khai

Thực hiện theo các lát dọc có thể kiểm chứng:

1. Design tokens và primitives dùng chung.
2. App shell, sidebar, utility header và navigation theo RBAC.
3. Dashboard riêng cho từng vai trò.
4. Danh sách/hàng đợi và filter/data table.
5. Form tạo/sửa phiếu và chi tiết phiếu.
6. Các module Finance, Operations, Audit, Admin và AI.
7. Responsive, accessibility, nội dung tiếng Việt và polish cuối.

Mỗi lát dọc phải có test trước hoặc đồng thời với thay đổi, chạy test/build liên quan và review trước khi chuyển sang lát tiếp theo.

## 13. Tiêu chí nghiệm thu

- Menu đúng theo role và không tràn ở desktop, tablet, mobile.
- Sidebar thu gọn/mở rộng hoạt động bằng chuột và bàn phím.
- Dashboard của sáu persona ưu tiên đúng hàng đợi nghiệp vụ.
- Các luồng tạo, sửa, gửi, phê duyệt, yêu cầu chỉnh sửa, từ chối, đặt hàng, giao nhận, hóa đơn và kiểm toán không bị thay đổi hành vi.
- Không có hành động trái quyền xuất hiện hoặc thực thi được.
- Bảng quan trọng không buộc người dùng mobile phải cuộn ngang để tìm hành động chính.
- Loading, empty, error và success state nhất quán.
- Unit/component tests hiện có tiếp tục đạt; bổ sung test navigation theo role và component mới.
- Frontend build thành công, không có lỗi TypeScript/template.
- Hoàn thành keyboard walkthrough, responsive review và accessibility audit trước khi tuyên bố hoàn tất.

## 14. Rủi ro và biện pháp giảm thiểu

| Rủi ro | Biện pháp |
| --- | --- |
| Thay shell làm sai menu role | Viết test ma trận role–navigation trước khi thay template |
| Refactor rộng gây lỗi luồng nghiệp vụ | Giữ service/state hiện có; thay đổi theo từng lát dọc nhỏ |
| Component abstraction quá mức | Chỉ trích xuất sau khi xác nhận mẫu lặp thực tế |
| Bảng đẹp trên desktop nhưng khó dùng mobile | Xác định ưu tiên cột theo từng màn hình và test reflow sớm |
| Màu trạng thái mất tương phản | Đo contrast và luôn kèm text/icon |
| Tài liệu hướng dẫn lệch giao diện mới | Cập nhật ảnh/tên vị trí điều hướng sau khi UI ổn định |

## 15. Quyết định đã chốt

- Dùng sidebar trái thu gọn làm điều hướng desktop chính.
- Dùng mobile sheet thay cho menu ngang cuộn.
- Tổ chức module theo nhóm nghiệp vụ và lọc theo RBAC.
- Dashboard được thiết kế riêng theo persona.
- Giảm card, gradient, shadow và heading quá lớn.
- Giữ nguyên stack Angular/Tailwind/Spartan và toàn bộ hợp đồng nghiệp vụ.
