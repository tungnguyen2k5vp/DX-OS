import type { ReactNode } from "react";
import clsx from "clsx";
import Link from "@docusaurus/Link";
import Layout from "@theme/Layout";
import Heading from "@theme/Heading";
import styles from "./index.module.css";

type PathCard = {
  eyebrow: string;
  title: string;
  description: string;
  to: string;
  action: string;
};

const paths: PathCard[] = [
  {
    eyebrow: "Người dùng nghiệp vụ",
    title: "Sử dụng theo vai trò",
    description:
      "Đăng nhập, tạo phiếu, phê duyệt, quản lý ngân sách và đối soát báo cáo theo đúng quyền.",
    to: "/huong-dan-su-dung",
    action: "Mở hướng dẫn",
  },
  {
    eyebrow: "Developer",
    title: "Phát triển Go + Angular",
    description:
      "Hiểu module backend, route frontend, REST contract, data model và chiến lược kiểm thử.",
    to: "/implementation/BACKEND_GO",
    action: "Đọc tài liệu kỹ thuật",
  },
  {
    eyebrow: "DevOps & QA",
    title: "Cài đặt và vận hành",
    description:
      "Dựng stack bằng Docker, tạo user, chạy smoke test, xem log và xử lý sự cố local.",
    to: "/bat-dau",
    action: "Bắt đầu cài đặt",
  },
];

const capabilities = [
  "Keycloak OIDC + PKCE",
  "Procurement hai cấp duyệt",
  "Budget reservation & commitment",
  "Nextcloud attachments",
  "Angular reporting",
  "Metabase read-only",
];

function PathwayCard({ path }: { path: PathCard }): ReactNode {
  return (
    <article className={styles.pathCard}>
      <p className={styles.eyebrow}>{path.eyebrow}</p>
      <Heading as="h3">{path.title}</Heading>
      <p>{path.description}</p>
      <Link className={styles.cardLink} to={path.to}>
        {path.action}
        <span aria-hidden="true"> →</span>
      </Link>
    </article>
  );
}

export default function Home(): ReactNode {
  return (
    <Layout
      title="Tài liệu DX-OS"
      description="Trung tâm tài liệu cài đặt, sử dụng, kiến trúc và vận hành DX-OS Lab."
    >
      <main>
        <section className={styles.hero} aria-labelledby="hero-title">
          <div className={styles.heroInner}>
            <div className={styles.heroCopy}>
              <p className={styles.kicker}>DX-OS KNOWLEDGE BASE</p>
              <Heading id="hero-title" as="h1">
                Hiểu hệ thống nhanh.
                <span> Vận hành đúng quyền.</span>
              </Heading>
              <p className={styles.lead}>
                Một điểm vào duy nhất cho tài liệu người dùng, cài đặt, kiến
                trúc, API và runbook của DX-OS Lab.
              </p>
              <div className={styles.heroActions}>
                <Link
                  className={clsx(
                    "button button--primary button--lg",
                    styles.primaryAction,
                  )}
                  to="/bat-dau"
                >
                  Cài đặt DX-OS
                </Link>
                <Link
                  className={clsx(
                    "button button--secondary button--lg",
                    styles.secondaryAction,
                  )}
                  to="/huong-dan-su-dung"
                >
                  Xem hướng dẫn sử dụng
                </Link>
              </div>
            </div>

            <aside className={styles.systemCard} aria-label="Tóm tắt hệ thống">
              <div className={styles.systemCardHeader}>
                <span className={styles.statusDot} aria-hidden="true" />
                <span>MVP đang hoạt động</span>
              </div>
              <dl className={styles.metrics}>
                <div>
                  <dt>Vai trò</dt>
                  <dd>6</dd>
                </div>
                <div>
                  <dt>Cấp duyệt</dt>
                  <dd>2</dd>
                </div>
                <div>
                  <dt>Nguồn tài liệu</dt>
                  <dd>1</dd>
                </div>
              </dl>
              <div className={styles.flow} aria-label="Luồng phê duyệt">
                <span>Nhân viên</span>
                <span aria-hidden="true">→</span>
                <span>Trưởng bộ phận</span>
                <span aria-hidden="true">→</span>
                <span>Tài chính</span>
              </div>
            </aside>
          </div>
        </section>

        <section className={styles.section} aria-labelledby="choose-path">
          <div className={styles.sectionHeading}>
            <p className={styles.eyebrow}>LỘ TRÌNH ĐỌC</p>
            <Heading id="choose-path" as="h2">
              Bắt đầu từ công việc của bạn
            </Heading>
            <p>Không cần đọc toàn bộ tài liệu để hoàn thành một tác vụ.</p>
          </div>
          <div className={styles.pathGrid}>
            {paths.map((path) => (
              <PathwayCard key={path.title} path={path} />
            ))}
          </div>
        </section>

        <section
          className={clsx(styles.section, styles.capabilitySection)}
          aria-labelledby="current-scope"
        >
          <div>
            <p className={styles.eyebrow}>PHẠM VI HIỆN TẠI</p>
            <Heading id="current-scope" as="h2">
              Những gì đã được triển khai
            </Heading>
            <p className={styles.scopeCopy}>
              Tài liệu luôn phân biệt chức năng đang chạy với hạng mục nằm trong
              roadmap. RAG và Agent chưa thuộc phiên bản hiện tại.
            </p>
          </div>
          <ul className={styles.capabilityList}>
            {capabilities.map((capability) => (
              <li key={capability}>
                <span aria-hidden="true">✓</span>
                {capability}
              </li>
            ))}
          </ul>
        </section>

        <section className={styles.cta} aria-labelledby="need-help">
          <div>
            <Heading id="need-help" as="h2">
              Gặp lỗi khi chạy local?
            </Heading>
            <p>
              Kiểm tra redirect URI, role, Docker health và các lỗi thường gặp
              trước khi thay đổi cấu hình.
            </p>
          </div>
          <Link
            className="button button--outline button--lg"
            to="/huong-dan-su-dung#14-xử-lý-sự-cố-cho-người-dùng"
          >
            Mở phần xử lý sự cố
          </Link>
        </section>
      </main>
    </Layout>
  );
}
