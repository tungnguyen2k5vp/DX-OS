import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';

interface GuideSection {
  number: string;
  title: string;
  route: string;
  linkLabel: string;
  purpose: string;
  tasks: string[];
  note?: string;
}

interface RequestStatusGuide {
  status: string;
  label: string;
  employeeAction: string;
}

@Component({
  selector: 'app-employee-guide',
  imports: [RouterLink, HlmBadge, HlmButton, ...HlmCardImports],
  templateUrl: './employee-guide.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EmployeeGuide {
  readonly workflow = [
    'Tạo và kiểm tra bản nháp',
    'Đính kèm tài liệu cần thiết',
    'Gửi phiếu cho Trưởng bộ phận',
    'Bổ sung nếu bị yêu cầu chỉnh sửa',
    'Theo dõi duyệt và đặt hàng',
    'Xác nhận khi đã nhận đủ hàng',
  ];

  readonly sections: GuideSection[] = [
    {
      number: '01',
      title: 'Tổng quan',
      route: '/dashboard',
      linkLabel: 'Mở Tổng quan',
      purpose: 'Xem nhanh tình hình phiếu và việc đang cần bạn xử lý.',
      tasks: [
        'Kiểm tra ô “Cần tôi bổ sung” để biết có phiếu bị trả lại hay không.',
        'Mở danh sách phiếu cập nhật gần đây để theo dõi trạng thái mới nhất.',
        'Dùng nút “Tạo phiếu mới” khi phát sinh nhu cầu mua sắm.',
      ],
    },
    {
      number: '02',
      title: 'Phiếu mua sắm',
      route: '/purchase-requests',
      linkLabel: 'Mở Phiếu mua sắm',
      purpose: 'Tạo, hoàn thiện, gửi và theo dõi yêu cầu mua sắm của chính bạn.',
      tasks: [
        'Chọn nhanh hàng hóa thường dùng từ danh mục; giá hiển thị là giá tham khảo và có thể sửa.',
        'Nhập tiêu đề, lý do từ 10 ký tự, trung tâm chi phí và ít nhất một dòng hàng.',
        'Kiểm tra số lượng, đơn giá và tổng tiền trước khi lưu bản nháp.',
        'Bấm “Kiểm tra phiếu trùng”; nếu có phiếu tương tự, mở phiếu đó để tránh mua lặp.',
        'Mở phiếu vừa lưu, tải tài liệu lên rồi bấm “Gửi phê duyệt”.',
        'Nếu tổng giá trị từ 20.000.000 VND, phải có tài liệu báo giá trước khi gửi.',
      ],
      note: 'Chỉ lưu bản nháp thì phiếu chưa vào hàng đợi phê duyệt của Trưởng bộ phận. Bạn phải mở chi tiết và bấm “Gửi phê duyệt”.',
    },
    {
      number: '03',
      title: 'Việc của tôi',
      route: '/work-center',
      linkLabel: 'Mở Việc của tôi',
      purpose: 'Tập trung các bản nháp và phiếu đang chờ bạn bổ sung.',
      tasks: [
        'Ưu tiên việc quá hạn xử lý hoặc sắp đến hạn được đưa lên đầu danh sách.',
        'Bấm “Mở công việc” để xem yêu cầu chỉnh sửa và nội dung trao đổi.',
        'Sửa thông tin, bổ sung tệp rồi chọn “Gửi lại” để chuyển phiếu về Trưởng bộ phận.',
      ],
    },
    {
      number: '04',
      title: 'Giao nhận',
      route: '/operations',
      linkLabel: 'Mở Giao nhận',
      purpose: 'Theo dõi đơn hàng sau khi phiếu được duyệt và xác nhận thực nhận.',
      tasks: [
        'Theo dõi nhà cung cấp, mã đơn hàng và ngày giao dự kiến.',
        'Khi hàng đã giao đủ và đúng, bấm “Xác nhận đã nhận”.',
        'Không xác nhận nếu hàng thiếu, sai hoặc hỏng; hãy trao đổi trên phiếu trước.',
      ],
      note: 'Nút xác nhận chỉ xuất hiện sau khi bộ phận Tài chính phát hành đơn hàng và tài khoản của bạn là người yêu cầu hoặc thuộc phạm vi được phép.',
    },
    {
      number: '05',
      title: 'Thông báo',
      route: '/notifications',
      linkLabel: 'Mở Thông báo',
      purpose: 'Không bỏ lỡ thay đổi trạng thái hoặc yêu cầu từ người xử lý.',
      tasks: [
        'Mở thông báo chưa đọc để đi thẳng tới phiếu liên quan.',
        'Kiểm tra thông báo sau mỗi lần Trưởng bộ phận hoặc bộ phận Tài chính xử lý phiếu.',
        'Đánh dấu đã đọc từng thông báo hoặc toàn bộ sau khi xử lý xong.',
      ],
    },
  ];

  readonly statuses: RequestStatusGuide[] = [
    {
      status: 'DRAFT',
      label: 'Bản nháp',
      employeeAction: 'Hoàn thiện nội dung, đính kèm tệp và gửi phê duyệt.',
    },
    {
      status: 'SUBMITTED',
      label: 'Đã gửi',
      employeeAction: 'Chờ Trưởng bộ phận duyệt; có thể theo dõi và trao đổi trên phiếu.',
    },
    {
      status: 'CHANGES_REQUESTED',
      label: 'Yêu cầu chỉnh sửa',
      employeeAction: 'Đọc lý do, sửa phiếu, bổ sung tệp và bấm “Gửi lại”.',
    },
    {
      status: 'MANAGER_APPROVED',
      label: 'Trưởng bộ phận đã duyệt',
      employeeAction: 'Chờ bộ phận Tài chính kiểm tra ngân sách và ra quyết định.',
    },
    {
      status: 'APPROVED',
      label: 'Đã phê duyệt',
      employeeAction: 'Theo dõi việc phát hành đơn hàng tại Giao nhận.',
    },
    {
      status: 'REJECTED / CANCELLED',
      label: 'Từ chối / Đã hủy',
      employeeAction: 'Đọc lý do trong lịch sử xử lý; phiếu đã kết thúc và không gửi lại được.',
    },
  ];

  readonly employeeCan = [
    'Tạo, sửa và hủy phiếu của mình khi trạng thái cho phép',
    'Chọn nhanh từ danh mục và kiểm tra nhu cầu trùng trước khi lưu',
    'Tải lên, tải xuống và xóa tệp do mình đính kèm khi còn được chỉnh sửa',
    'Gửi phiếu, gửi lại phiếu và trao đổi với người xử lý',
    'Theo dõi lịch sử xử lý, ngân sách khả dụng và trạng thái giao nhận',
    'Xác nhận đã nhận hàng đối với đơn hàng hợp lệ',
  ];

  readonly employeeCannot = [
    'Phê duyệt thay Trưởng bộ phận hoặc bộ phận Tài chính',
    'Điều chỉnh hạn mức ngân sách',
    'Chọn nhà cung cấp hoặc phát hành đơn hàng',
    'Xử lý hóa đơn, thanh toán, chính sách hay báo cáo kiểm toán',
  ];
}
