---
id: user-guide
title: Hướng dẫn sử dụng DX-OS Lab
description: Hướng dẫn toàn bộ quy trình nghiệp vụ và quyền của sáu role.
slug: /huong-dan-su-dung
sidebar_position: 3
---

# Hướng dẫn sử dụng DX-OS Lab

Tài liệu này hướng dẫn sử dụng phiên bản hiện tại của DX-OS Lab từ lúc đăng nhập đến khi một phiếu
mua sắm được duyệt, đồng thời giải thích quyền của sáu role trong realm Keycloak **dx-os**.

## 1. Phạm vi hiện tại

Người dùng có thể:

- đăng nhập bằng tài khoản do Keycloak quản lý;
- tạo và theo dõi phiếu mua sắm;
- xử lý quy trình duyệt hai cấp;
- kiểm tra, giữ, cam kết và điều chỉnh ngân sách theo quyền;
- tải lên, tải xuống và kiểm soát tài liệu hồ sơ;
- xem báo cáo vận hành trên DX-OS hoặc phân tích sâu trong Metabase.

RAG và AI Agent chưa có màn hình hoặc quy trình sử dụng trong phiên bản này. Role ai_operator đã
được khai báo để chuẩn bị cho giai đoạn đó, không nên hiểu là tính năng AI đã hoàn thành.

## 2. Đăng nhập và đăng xuất

### Đăng nhập

1. Mở http://localhost:4200.
2. Ứng dụng chuyển sang trang đăng nhập Keycloak của realm dx-os.
3. Nhập username và password của tài khoản được cấp.
4. Sau khi xác thực thành công, Keycloak chuyển về trang Tổng quan của DX-OS.

Chỉ sử dụng hostname **localhost** trong môi trường hiện tại. Nếu mở bằng 127.0.0.1, Keycloak có
thể báo Invalid parameter: redirect_uri vì URI đó không nằm trong allowlist của client dx-web.

### Đăng xuất

Nhấn **Đăng xuất** ở góc trên bên phải. DX-OS kết thúc phiên Keycloak và xóa trạng thái đăng nhập
cục bộ. Không chỉ đóng tab khi dùng máy dùng chung.

### Tài khoản local

Hệ thống có sáu **role**, không phải chỉ có sáu account cố định. Quản trị viên có thể tạo nhiều
tài khoản và gán role phù hợp. Bộ account demo nên tách riêng như sau:

| Username gợi ý   | Role               | Mục đích kiểm thử                     |
| ---------------- | ------------------ | ------------------------------------- |
| employee.demo    | employee           | Tạo, sửa và gửi phiếu của mình        |
| manager.demo     | department_manager | Duyệt phiếu của phòng ban             |
| finance.demo     | finance            | Duyệt cuối và quản lý ngân sách       |
| auditor.demo     | auditor            | Đọc hồ sơ, ngân sách và báo cáo       |
| ai.operator.demo | ai_operator        | Dành cho giai đoạn AI sau này         |
| admin.demo       | dx_admin           | Truy cập báo cáo và cấu hình được cấp |

Cách tạo tài khoản và vị trí file mật khẩu được mô tả trong [Bắt đầu với DX-OS](GETTING_STARTED.md). Sau khi role
của một user thay đổi, user phải đăng xuất rồi đăng nhập lại để access token chứa role mới.

## 3. Role và quyền hiện hành

| Chức năng                | Employee |     Manager     |        Finance         |     Auditor      |   AI operator   |     DX admin      |
| ------------------------ | :------: | :-------------: | :--------------------: | :--------------: | :-------------: | :---------------: |
| Tổng quan                |    Có    |       Có        |           Có           |        Có        |       Có        |        Có         |
| Tạo phiếu                |    Có    |       Có        |         Không          |      Không       |      Không      |       Không       |
| Sửa/gửi phiếu của mình   |    Có    |       Có        |         Không          |      Không       |      Không      |       Không       |
| Xem phiếu                | Của mình | Trong phòng ban | Theo phạm vi tài chính | Toàn bộ, chỉ đọc |    Chưa cấp     | Chưa cấp mặc định |
| Duyệt cấp trưởng bộ phận |  Không   | Đúng phòng ban  |         Không          |      Không       |      Không      |       Không       |
| Duyệt cấp tài chính      |  Không   |      Không      |           Có           |      Không       |      Không      |       Không       |
| Dashboard ngân sách      |  Không   |      Không      |      Có, được sửa      |   Có, chỉ đọc    |      Không      |       Không       |
| Báo cáo DX-OS            |  Không   |      Không      |     Trong tổ chức      |     Toàn bộ      |      Không      |      Toàn bộ      |
| Tính năng AI             | Chưa có  |     Chưa có     |        Chưa có         |     Chưa có      | Chưa triển khai |      Chưa có      |

Lưu ý quan trọng:

- Quyền thực được kiểm tra tại Go API; việc ẩn menu trên Angular không phải hàng rào bảo mật.
- Manager chỉ duyệt phiếu đúng phòng ban và không được tự duyệt phiếu do mình tạo.
- Finance xử lý bước tài chính trong phạm vi tổ chức/cost center được cấp.
- Auditor có quyền đọc rộng để đối soát nhưng không được thay đổi dữ liệu.
- dx_admin không phải superuser nghiệp vụ, không mặc nhiên được xem hoặc duyệt mọi phiếu.
- Nếu một user có nhiều role, backend áp dụng data scope theo policy. Khi kiểm thử nên dùng account
  tách biệt để tránh nhầm quyền.

## 4. Menu và trang

| Menu/đường dẫn               | Nội dung                                     | Ai sử dụng                          |
| ---------------------------- | -------------------------------------------- | ----------------------------------- |
| /dashboard                   | Tổng quan và trạng thái kết nối              | Mọi user đã đăng nhập               |
| /purchase-requests           | Danh sách phiếu trong data scope             | Employee, manager, finance, auditor |
| /purchase-requests/new       | Form tạo phiếu                               | Employee, manager                   |
| /purchase-requests/{id}      | Chi tiết, hành động, tệp và timeline         | User có quyền xem phiếu             |
| /purchase-requests/{id}/edit | Sửa draft hoặc phiếu bị trả sửa              | Chủ phiếu là employee/manager       |
| /approvals                   | Hộp thư chờ duyệt                            | Manager, finance                    |
| /budgets                     | Hạn mức, cảnh báo, reservation và điều chỉnh | Finance, auditor                    |
| /reports                     | KPI vận hành                                 | Finance, auditor, dx_admin          |

Menu được hiển thị theo role. Nếu nhập trực tiếp một URL không được phép, Angular sẽ chuyển hướng
hoặc Go API trả HTTP 403.

## 5. Vòng đời phiếu mua sắm

```text
DRAFT
  | submit
  v
SUBMITTED
  | manager approve
  v
MANAGER_APPROVED
  | finance approve
  v
APPROVED

Ở hai bước duyệt: có thể REJECTED hoặc CHANGES_REQUESTED.
DRAFT và CHANGES_REQUESTED: chủ phiếu có thể CANCELLED.
CHANGES_REQUESTED: chủ phiếu sửa rồi resubmit về SUBMITTED.
```

| Trạng thái              | Ý nghĩa                          | Người cần hành động  |
| ----------------------- | -------------------------------- | -------------------- |
| Bản nháp                | Phiếu chưa gửi                   | Chủ phiếu            |
| Đã gửi                  | Chờ duyệt cấp phòng ban          | Department manager   |
| Trưởng bộ phận đã duyệt | Chờ kiểm tra/duyệt tài chính     | Finance              |
| Yêu cầu chỉnh sửa       | Bị trả lại để bổ sung            | Chủ phiếu            |
| Đã duyệt                | Hoàn thành hai cấp duyệt         | Không còn bước duyệt |
| Từ chối                 | Quy trình kết thúc do bị từ chối | Không còn bước duyệt |
| Đã hủy                  | Chủ phiếu chủ động dừng          | Không còn bước duyệt |

Mỗi thay đổi trạng thái tạo một process event. Trang chi tiết hiển thị timeline gồm thời gian,
người thực hiện, role, trạng thái và nhận xét.

## 6. Hướng dẫn cho Employee

### 6.1 Tạo phiếu

1. Chọn **Phiếu mua sắm**.
2. Nhấn **Tạo phiếu mới**.
3. Nhập tiêu đề, lý do, cost center và tiền tệ.
4. Thêm ít nhất một dòng hàng: nội dung, số lượng, đơn giá và thông tin liên quan.
5. Kiểm tra tổng tiền rồi lưu.

Các nguyên tắc chính:

- Lý do từ 10 đến 5.000 ký tự.
- Cost center tối đa 100 ký tự.
- Một phiếu có từ 1 đến 100 dòng hàng.
- Tổng tiền được backend tính lại; không tin tổng tiền do trình duyệt gửi lên.
- Phiếu mới luôn bắt đầu ở trạng thái Bản nháp.

### 6.2 Sửa phiếu

Chủ phiếu chỉ sửa được khi trạng thái là **Bản nháp** hoặc **Yêu cầu chỉnh sửa**:

1. Mở mã phiếu trong danh sách.
2. Chọn **Chỉnh sửa**.
3. Cập nhật thông tin và lưu.

Nếu người khác đã cập nhật cùng phiên bản, hệ thống có thể báo xung đột. Tải lại trang, xem dữ liệu
mới nhất rồi thực hiện lại thay đổi; không cố ghi đè mù.

### 6.3 Đính kèm tài liệu

Trong trang chi tiết, tìm khu vực **Tài liệu đính kèm**:

1. Chọn loại tài liệu: Báo giá, Đặc tả, Hợp đồng hoặc Khác.
2. Chọn tệp rồi tải lên.
3. Kiểm tra tên tệp, kích thước và loại tài liệu trong danh sách.

Loại tệp cho phép: PDF, DOCX, XLSX, JPG/JPEG và PNG. Dung lượng tối đa là 10 MB/tệp. Chỉ chủ
phiếu được tải lên hoặc xóa tệp khi phiếu đang là Bản nháp/Yêu cầu chỉnh sửa. Người có quyền xem
phiếu có thể tải xuống tài liệu.

Nếu tổng phiếu từ **20.000.000 VND**, phải có ít nhất một tệp loại **Báo giá** còn hiệu lực trước
khi gửi hoặc gửi lại. Việc chỉ tải một tệp loại Khác không thỏa điều kiện này.

### 6.4 Kiểm tra và gửi duyệt

1. Mở trang chi tiết phiếu.
2. Xem kết quả kiểm tra ngân sách và yêu cầu tài liệu.
3. Nhấn **Gửi duyệt**.
4. Xác nhận trạng thái chuyển sang **Đã gửi**.

Nếu submit thất bại, đọc thông báo cụ thể: thường do thiếu báo giá, dữ liệu không hợp lệ, phiếu đã
đổi phiên bản hoặc tài khoản không còn quyền.

### 6.5 Khi phiếu bị trả sửa

1. Mở timeline để đọc nhận xét bắt buộc của người duyệt.
2. Chọn **Chỉnh sửa**, bổ sung nội dung/tài liệu.
3. Lưu và nhấn **Gửi lại**.

Phiếu trở lại trạng thái Đã gửi để manager duyệt lại. Chủ phiếu cũng có thể **Hủy** nếu không tiếp
tục nhu cầu.

## 7. Hướng dẫn cho Department Manager

Manager có thể tạo phiếu như employee, đồng thời xử lý phiếu của phòng ban:

1. Chọn menu **Phê duyệt**.
2. Hộp thư hiển thị các phiếu ở trạng thái Đã gửi thuộc đúng department.
3. Mở phiếu, đọc nội dung, dòng hàng, tài liệu, budget check và timeline.
4. Chọn một hành động:
   - **Phê duyệt**: chuyển sang bước tài chính và giữ số tiền ngân sách;
   - **Yêu cầu chỉnh sửa**: trả phiếu cho chủ sở hữu;
   - **Từ chối**: kết thúc quy trình.
5. Nhập nhận xét khi yêu cầu chỉnh sửa hoặc từ chối.

Manager không được tự duyệt phiếu mình tạo. Nếu ngân sách không đủ, bước phê duyệt không được
commit. Sau khi xử lý thành công, phiếu rời khỏi hộp thư chờ duyệt của manager.

## 8. Hướng dẫn cho Finance

### 8.1 Duyệt cuối

1. Chọn **Phê duyệt**.
2. Danh sách hiển thị phiếu đã được manager duyệt.
3. Đối chiếu cost center, số tiền, tài liệu và phần ngân sách đang được giữ.
4. Chọn:
   - **Phê duyệt** để chuyển reservation thành committed amount;
   - **Yêu cầu chỉnh sửa** để trả chủ phiếu và giải phóng reservation;
   - **Từ chối** để kết thúc và giải phóng reservation.

Yêu cầu chỉnh sửa/từ chối phải có nhận xét rõ ràng. Finance cũng không được tự duyệt phiếu của
chính mình trong trường hợp tài khoản có thêm quyền tạo.

### 8.2 Quản lý ngân sách

Chọn **Ngân sách** để xem:

- kỳ ngân sách, cost center và currency;
- allocated amount, reserved amount, committed amount và available amount;
- tỷ lệ sử dụng và mức cảnh báo;
- các reservation đang giữ tiền;
- lịch sử giao dịch và lịch sử điều chỉnh.

Để thay đổi tổng hạn mức:

1. Chọn **Điều chỉnh** trên allocation.
2. Nhập hạn mức mới.
3. Nhập lý do từ 10 đến 1.000 ký tự.
4. Xác nhận và kiểm tra dòng audit mới.

Không giảm hạn mức xuống dưới tổng reserved + committed. Nếu dữ liệu vừa bị người khác cập nhật,
tải lại dashboard trước khi thử lại.

### 8.3 Xem báo cáo

Chọn **Báo cáo**. Finance xem dữ liệu trong organization của mình và có thể lọc theo:

- từ ngày, đến ngày;
- department;
- cost center;
- currency.

Không cộng các currency khác nhau thành một tổng. Khi đối soát, luôn ghi lại bộ lọc đang dùng.

## 9. Hướng dẫn cho Auditor

Auditor dùng hệ thống ở chế độ đọc:

- xem phiếu và timeline theo mandate;
- tải tài liệu bằng đường đi qua Go API;
- xem dashboard Ngân sách nhưng không có nút Điều chỉnh;
- xem báo cáo toàn bộ phạm vi được cấp;
- đối chiếu actor, role, timestamp, comment và các event ngân sách.

Nếu auditor thấy nút thay đổi dữ liệu hoặc một lệnh ghi thành công, coi đó là lỗi phân quyền và báo
ngay cho nhóm phát triển. Không dùng tài khoản auditor để làm nghiệp vụ thay finance.

## 10. Hướng dẫn cho DX Admin

Trong phiên bản hiện tại, dx_admin có thể vào **Báo cáo** với phạm vi toàn bộ dữ liệu báo cáo.
Role này không mặc nhiên:

- tạo/sửa phiếu;
- duyệt cấp manager hoặc finance;
- điều chỉnh ngân sách;
- đọc mọi hồ sơ nghiệp vụ như auditor.

Quản trị realm/user trong local được thực hiện qua Keycloak Admin Console hoặc script vận hành,
không phải qua một trang quản trị DX-OS riêng. Mọi quyền support mở rộng sau này phải có endpoint,
policy và audit riêng; không gán role nghiệp vụ rộng chỉ để xử lý nhanh.

## 11. Hướng dẫn cho AI Operator

ai_operator là role dự trữ cho giai đoạn RAG/Agent. Hiện tài khoản này chỉ xác thực và vào trang
Tổng quan; chưa có menu AI, recommendation, approval queue hoặc tool execution.

Khi giai đoạn Agent được triển khai, nguyên tắc đã chốt là:

- AI chỉ đề xuất, không tự cấp quyền cho mình;
- tool phải nằm trong allowlist của backend;
- thao tác nhạy cảm cần người có quyền phê duyệt;
- input, output, người phê duyệt và kết quả thực thi phải được audit.

Không dùng role ai_operator để thay thế manager hoặc finance trong phiên bản hiện tại.

## 12. Báo cáo DX-OS và Metabase khác nhau thế nào?

### Báo cáo trong DX-OS

- URL: http://localhost:4200/reports
- Đăng nhập bằng Keycloak.
- Giao diện KPI cố định, phù hợp theo dõi hàng ngày.
- Go API áp dụng data scope cho finance, auditor và dx_admin.

### Metabase

- URL: http://localhost:3000
- Dùng tài khoản Metabase riêng tại data/runtime/metabase-admin.txt.
- Phục vụ phân tích ad-hoc và chỉnh dashboard.
- Data source dùng role dxos_report_reader, chỉ đọc schema reporting.
- Dashboard DX-OS Procurement có 8 card và bộ lọc từ ngày, đến ngày, tiền tệ.

Một card hiển thị **0** không nhất thiết là lỗi: đó có thể là KPI dạng số và tập dữ liệu hiện không
có dòng phù hợp bộ lọc. Hãy bỏ/broaden bộ lọc, tạo dữ liệu đúng trạng thái rồi chạy lại báo cáo.

## 13. Kịch bản kiểm thử toàn bộ quy trình

Dùng ba cửa sổ trình duyệt riêng hoặc đăng xuất giữa các bước:

1. Employee tạo phiếu, thêm dòng hàng và lưu draft.
2. Nếu tổng từ 20 triệu VND, employee tải lên một Báo giá hợp lệ.
3. Employee gửi duyệt; trạng thái thành Đã gửi.
4. Manager mở Phê duyệt và duyệt; trạng thái thành Trưởng bộ phận đã duyệt, ngân sách chuyển sang
   reserved.
5. Finance mở Phê duyệt và duyệt; trạng thái thành Đã duyệt, ngân sách reserved giảm và committed
   tăng.
6. Auditor mở phiếu/timeline, Ngân sách và Báo cáo để đối chiếu nhưng không thay đổi dữ liệu.
7. Mở Metabase, đặt đúng khoảng ngày/currency và đối chiếu số liệu với DX-OS.

Nên chạy thêm các smoke test trong README để xác nhận cả positive và negative authorization path.

## 14. Xử lý sự cố cho người dùng

| Hiện tượng                   | Cách xử lý                                                                       |
| ---------------------------- | -------------------------------------------------------------------------------- |
| redirect_uri không hợp lệ    | Mở lại http://localhost:4200; không dùng 127.0.0.1                               |
| Sai username/password        | Nhờ tạo lại user local; mật khẩu đổi mỗi lần script chạy                         |
| Đổi role nhưng menu chưa đổi | Đăng xuất và đăng nhập lại để nhận token mới                                     |
| 401 Unauthorized             | Phiên hết hạn/không hợp lệ; đăng nhập lại                                        |
| 403 Forbidden                | Role hoặc data scope không cho phép; không phải lỗi mật khẩu                     |
| Không submit được phiếu      | Kiểm tra trạng thái, báo giá, dữ liệu và phiên bản phiếu                         |
| Không duyệt được             | Kiểm tra đúng bước, đúng department/organization, không tự duyệt và đủ ngân sách |
| Không thấy menu Ngân sách    | Chỉ finance/auditor có quyền; đăng nhập lại sau khi gán role                     |
| Không thấy menu Báo cáo      | Chỉ finance/auditor/dx_admin có quyền                                            |
| Metabase hiện 0              | Kiểm tra khoảng ngày, currency và dữ liệu đã phát sinh                           |

Khi báo lỗi cho nhóm phát triển, cung cấp: thời điểm, username (không gửi password), role, mã phiếu,
URL, thao tác vừa làm và nội dung thông báo. Không chụp hoặc gửi access token/secret.

## 15. Quy tắc an toàn

- Không chia sẻ mật khẩu, file credential, .env hoặc token.
- Không tải tài liệu thật có dữ liệu nhạy cảm lên môi trường lab.
- Không sửa trực tiếp PostgreSQL để “chữa” trạng thái phiếu.
- Không đăng nhập Nextcloud bằng service account của backend.
- Không cấp thêm role rộng để vượt qua lỗi 403; cần xác định đúng policy/data scope.
- Luôn đăng xuất khi dùng máy chung.

## 16. Tài liệu liên quan

- [Bắt đầu và cài đặt](GETTING_STARTED.md)
- [Authentication và Authorization](implementation/AUTHORIZATION.md)
- [Procurement MVP runbook](runbooks/PROCUREMENT_MVP.md)
- [Attachment runbook](runbooks/ATTACHMENTS.md)
- [Reporting runbook](runbooks/REPORTING.md)
- OpenAPI contract trong source: `contracts/openapi/dx-os-v1.yaml`.
