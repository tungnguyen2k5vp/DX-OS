# DX-OS Enterprise Operations Shell Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Thay giao diện DX-OS bằng Enterprise Operations Shell có sidebar theo RBAC, dashboard riêng theo vai trò và mẫu trang tác nghiệp nhất quán trên desktop/mobile mà không đổi nghiệp vụ.

**Architecture:** Tách quyết định điều hướng khỏi root component thành một navigation model thuần có thể kiểm thử; root `App` chỉ phối hợp auth, notification và trạng thái shell. Dùng các lớp component CSS dùng chung cho page header, metric strip, data panel, filter bar và responsive table, sau đó chuyển từng nhóm màn hình theo lát dọc để luôn giữ được một frontend chạy được.

**Tech Stack:** Angular 22 standalone components, TypeScript 6, Tailwind CSS 4, Spartan HLM primitives, Vitest 4.

**Spec:** `docs/superpowers/specs/2026-08-18-enterprise-operations-shell-design.md`

## Global Constraints

- Giữ nguyên Angular 22, Tailwind CSS 4, Spartan và cấu trúc standalone component hiện tại.
- Không thêm React, package shadcn/ui hoặc thay backend/database.
- Giữ nguyên URL/route, Keycloak, role guards, API contracts, workflow và RBAC hiện có.
- Không hiển thị hành động ngoài quyền và không suy quyền từ trạng thái UI lưu cục bộ.
- Toàn bộ nội dung người dùng phổ thông phải là tiếng Việt; giữ SLA, API, AI và RBAC khi đúng ngữ cảnh.
- Mục tiêu accessibility là WCAG 2.2 AA nhưng không tuyên bố đạt trước khi có kiểm thử thủ công.
- Không commit, push, merge hoặc thay đổi `main` khi chưa có yêu cầu riêng của người dùng.
- Không chạm vào thay đổi có sẵn `docs/generated/So_tay_kiem_thu_DX_OS_theo_vai_tro_chi_tiet.docx`.
- Với code có hành vi: bắt buộc RED → xác nhận lỗi đúng nguyên nhân → GREEN → chạy lại test. CSS thuần được kiểm chứng bằng build, responsive walkthrough và accessibility review thay vì test kiểm tra chuỗi class.
- Quy ước lệnh: mọi lệnh `npm ...` chạy với working directory `DX-OS/frontend`; mọi lệnh `git ...` và tìm kiếm toàn repo chạy với working directory `DX-OS`.
- Mỗi data page phải có loading, empty, error và success state; disabled state phải có lý do khi trang có điều khiển chưa thể dùng. Task sở hữu trang phải kiểm thử các state có hành vi, Task 10 kiểm kê chéo toàn bộ.

---

## File Structure

### Files created

- `frontend/src/app/core/navigation/navigation.model.ts`: type, label và hàm dựng navigation theo role.
- `frontend/src/app/core/navigation/navigation.model.spec.ts`: ma trận role–route và quy tắc ẩn nhóm rỗng.
- `frontend/src/app/core/layout/shell-state.service.ts`: trạng thái sidebar/mobile sheet và local storage adapter.
- `frontend/src/app/core/layout/shell-state.service.spec.ts`: hành vi mở/đóng, thu gọn và khôi phục trạng thái.
- `frontend/src/app/shared/ui/page-header/page-header.ts`: component header dùng chung với title, description và action slot.
- `frontend/src/app/shared/ui/page-header/page-header.spec.ts`: semantic heading và projection.
- `frontend/src/app/shared/ui/page-header/index.ts`: public export.
- `frontend/src/app/shared/ui/data-table/data-table-shell.ts`: wrapper semantic/scroll cho bảng tác nghiệp.
- `frontend/src/app/shared/ui/data-table/data-table-shell.spec.ts`: accessible name và table region behavior.
- `frontend/src/app/shared/ui/data-table/index.ts`: public export.
- `frontend/src/app/features/procurement/pages/purchase-request-detail/purchase-request-detail.spec.ts`: regression test cho action panel theo role/trạng thái.
- `frontend/src/app/features/procurement/pages/purchase-request-list/purchase-request-list.spec.ts`: state matrix và hành động chính của danh sách phiếu.
- `frontend/src/app/features/admin/pages/admin-center/admin-center.spec.ts`: state và task hierarchy của Admin Center.
- `frontend/src/app/features/ai/pages/recommendation-center/recommendation-center.spec.ts`: state và explainability của AI recommendations.

### Files modified

- `frontend/src/app/app.ts`, `app.html`, `app.css`, `app.spec.ts`: Enterprise App Shell.
- `frontend/src/styles.css`: tokens, typography, flat sections, responsive data patterns và reduced motion.
- `frontend/src/app/shared/ui/card/src/lib/hlm-card.ts`: card phẳng hơn, radius/shadow theo spec.
- `frontend/src/app/shared/ui/button/src/lib/hlm-button.ts`: hierarchy và motion tiết chế.
- `frontend/src/app/shared/ui/badge/src/lib/hlm-badge.ts`: status chip dễ đọc.
- Các page template/spec dưới `features/dashboard`, `features/procurement`, `features/reporting`, `features/admin`, `features/ai`: chuyển sang page patterns mới mà không đổi service calls.

---

### Task 1: Navigation model theo RBAC

**Files:**
- Create: `frontend/src/app/core/navigation/navigation.model.ts`
- Create: `frontend/src/app/core/navigation/navigation.model.spec.ts`

**Interfaces:**
- Produces: `AppRole`, `NavigationItem`, `NavigationGroup`, `navigationForRoles(roles: readonly string[]): NavigationGroup[]`, `primaryRoleLabel(roles: readonly string[]): string`.
- Consumed by: Task 3 root App Shell and Task 4 dashboard context.

- [ ] **Step 1: Write the failing role matrix tests**

```ts
import { navigationForRoles, primaryRoleLabel } from './navigation.model';

const routes = (roles: string[]) =>
  navigationForRoles(roles).flatMap((group) => group.items.map((item) => item.route));

describe('navigationForRoles', () => {
  it('gives employees only their operational routes', () => {
    expect(routes(['employee'])).toEqual([
      '/dashboard', '/work-center', '/notifications', '/purchase-requests',
      '/operations', '/employee-guide',
    ]);
  });

  it('does not give auditors mutating approval or administration routes', () => {
    const auditorRoutes = routes(['auditor']);
    expect(auditorRoutes).toContain('/audit');
    expect(auditorRoutes).toContain('/budgets');
    expect(auditorRoutes).not.toContain('/approvals');
    expect(auditorRoutes).not.toContain('/admin');
  });

  it('removes navigation groups that have no authorized items', () => {
    expect(navigationForRoles(['employee']).every((group) => group.items.length > 0)).toBe(true);
  });

  it('uses the first recognized role for the role label', () => {
    expect(primaryRoleLabel(['finance', 'auditor'])).toBe('Tài chính');
    expect(primaryRoleLabel(['unknown'])).toBe('Đã xác thực');
  });
});
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `npm test -- --watch=false --include=src/app/core/navigation/navigation.model.spec.ts`

Expected: FAIL because `navigation.model.ts` and exported functions do not exist.

- [ ] **Step 3: Implement the typed navigation catalog and filtering**

```ts
export type AppRole =
  | 'employee' | 'department_manager' | 'finance'
  | 'auditor' | 'dx_admin' | 'ai_operator';

export interface NavigationItem {
  readonly label: string;
  readonly route: string;
  readonly shortLabel: string;
  readonly roles: readonly AppRole[];
}

export interface NavigationGroup {
  readonly label: string;
  readonly items: readonly NavigationItem[];
}

export function navigationForRoles(roles: readonly string[]): NavigationGroup[] {
  const granted = new Set(roles);
  return NAVIGATION_GROUPS
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => item.roles.some((role) => granted.has(role))),
    }))
    .filter((group) => group.items.length > 0);
}
```

Catalog routes and labels exactly match the approved information architecture. `/employee-guide` is an employee contextual-help item placed after `/operations` so the literal employee expectation remains stable.

- [ ] **Step 4: Run focused test and verify GREEN**

Run: `npm test -- --watch=false --include=src/app/core/navigation/navigation.model.spec.ts`

Expected: 4 tests pass, 0 failures.

- [ ] **Step 5: Review the uncommitted diff for route/RBAC drift**

Run: `git diff -- frontend/src/app/core/navigation`

Check every route against `app.routes.ts` and every role against existing guards. Do not commit.

### Task 2: Shell state service

**Files:**
- Create: `frontend/src/app/core/layout/shell-state.service.ts`
- Create: `frontend/src/app/core/layout/shell-state.service.spec.ts`

**Interfaces:**
- Produces: `ShellStorage`, `SHELL_STORAGE`, `ShellState` with `sidebarCollapsed`, `mobileNavigationOpen`, `toggleSidebar()`, `openMobileNavigation()`, `closeMobileNavigation()`.
- Consumed by: Task 3 `App`.

- [ ] **Step 1: Write failing behavior tests using an in-memory storage adapter**

```ts
describe('ShellState', () => {
  it('restores the collapsed preference and persists toggles', () => {
    const storage = memoryStorage({ 'dx-os.sidebar-collapsed': 'true' });
    TestBed.configureTestingModule({ providers: [{ provide: SHELL_STORAGE, useValue: storage }] });
    const state = TestBed.inject(ShellState);
    expect(state.sidebarCollapsed()).toBe(true);
    state.toggleSidebar();
    expect(state.sidebarCollapsed()).toBe(false);
    expect(storage.getItem('dx-os.sidebar-collapsed')).toBe('false');
  });

  it('opens and closes mobile navigation without changing the sidebar preference', () => {
    const state = TestBed.inject(ShellState);
    state.openMobileNavigation();
    expect(state.mobileNavigationOpen()).toBe(true);
    state.closeMobileNavigation();
    expect(state.mobileNavigationOpen()).toBe(false);
  });
});
```

- [ ] **Step 2: Run focused test and verify RED**

Run: `npm test -- --watch=false --include=src/app/core/layout/shell-state.service.spec.ts`

Expected: FAIL because the service and token do not exist.

- [ ] **Step 3: Implement minimal signal-based state**

Use an injection token whose browser default delegates to `localStorage`; guard read/write failures so private browsing/storage denial does not break the shell. Mobile state is session-only and defaults to closed.

- [ ] **Step 4: Run focused test and verify GREEN**

Run: `npm test -- --watch=false --include=src/app/core/layout/shell-state.service.spec.ts`

Expected: all shell-state tests pass with clean output.

- [ ] **Step 5: Run Task 1 and Task 2 tests together**

Run: `npm test -- --watch=false --include=src/app/core/navigation/navigation.model.spec.ts --include=src/app/core/layout/shell-state.service.spec.ts`

Expected: all tests pass; do not commit.

### Task 3: Enterprise App Shell

**Files:**
- Modify: `frontend/src/app/app.ts`
- Modify: `frontend/src/app/app.html`
- Modify: `frontend/src/app/app.css`
- Modify: `frontend/src/app/app.spec.ts`

**Interfaces:**
- Consumes: `navigationForRoles`, `primaryRoleLabel`, `ShellState`.
- Produces: desktop sidebar, mobile sheet, utility header, skip link and `#main-content` landmark for all routed pages.

- [ ] **Step 1: Replace old App tests with failing behavior tests**

Add assertions against rendered behavior, not class strings:

```ts
it('renders grouped employee navigation and a main-content skip target', () => {
  const fixture = TestBed.createComponent(App);
  fixture.detectChanges();
  const element = fixture.nativeElement as HTMLElement;
  expect(element.querySelector('a[href="#main-content"]')?.textContent).toContain('Bỏ qua');
  expect(element.querySelector('nav[aria-label="Điều hướng chính"]')).toBeTruthy();
  expect(element.querySelector('a[href="/employee-guide"]')).toBeTruthy();
  expect(element.querySelector('a[href="/admin"]')).toBeNull();
  expect(element.querySelector('main#main-content')).toBeTruthy();
});

it('opens and closes the mobile navigation as a dialog', () => {
  const fixture = TestBed.createComponent(App);
  fixture.detectChanges();
  const root = fixture.nativeElement as HTMLElement;
  (root.querySelector('[data-mobile-menu-trigger]') as HTMLButtonElement).click();
  fixture.detectChanges();
  expect(root.querySelector('[role="dialog"][aria-modal="true"]')).toBeTruthy();
  (root.querySelector('[data-mobile-menu-close]') as HTMLButtonElement).click();
  fixture.detectChanges();
  expect(root.querySelector('[role="dialog"]')).toBeNull();
});
```

Retain the existing logout/notification service test setup and expand the finance role assertion to ensure `/budgets` appears while `/employee-guide` does not.

- [ ] **Step 2: Run App tests and verify RED**

Run: `npm test -- --watch=false --include=src/app/app.spec.ts`

Expected: FAIL because the skip link, grouped nav, mobile dialog and shell state are not wired.

- [ ] **Step 3: Implement the minimal root coordination**

`App` exposes computed `navigationGroups` from auth roles, delegates shell actions to `ShellState`, and removes scattered `canAccess...` computed properties. It keeps notification refresh and logout behavior unchanged.

- [ ] **Step 4: Build the desktop sidebar and utility header**

The template contains:

```html
<a class="skip-link" href="#main-content">Bỏ qua điều hướng, đến nội dung chính</a>
<aside class="app-sidebar" [attr.data-collapsed]="shell.sidebarCollapsed()">
  <nav aria-label="Điều hướng chính">
    @for (group of navigationGroups(); track group.label) { ... }
  </nav>
</aside>
<div class="app-workspace">
  <header class="utility-header">...</header>
  <main id="main-content" tabindex="-1"><router-outlet /></main>
</div>
```

Use text plus compact inline SVG icons with `aria-hidden="true"`; do not add an icon dependency. Active route uses `aria-current="page"` through `RouterLinkActive` state.

- [ ] **Step 5: Build the mobile modal sheet**

Render only while `mobileNavigationOpen()` is true. The trigger and close controls have accessible names. Clicking a route or backdrop closes the sheet. Escape handling is implemented at `App` level; focus returns to the trigger after close. Use Angular CDK `FocusTrap` only if already available through `@angular/cdk`; do not add a new package.

- [ ] **Step 6: Run App tests and verify GREEN**

Run: `npm test -- --watch=false --include=src/app/app.spec.ts`

Expected: all App shell tests pass.

- [ ] **Step 7: Run navigation and shell state regression tests**

Run: `npm test -- --watch=false --include=src/app/core/navigation/navigation.model.spec.ts --include=src/app/core/layout/shell-state.service.spec.ts --include=src/app/app.spec.ts`

Expected: all tests pass, no warnings. Review `git diff --check`; do not commit.

### Task 4: Design tokens and shared page primitives

**Files:**
- Modify: `frontend/src/styles.css`
- Modify: `frontend/src/app/shared/ui/card/src/lib/hlm-card.ts`
- Modify: `frontend/src/app/shared/ui/button/src/lib/hlm-button.ts`
- Modify: `frontend/src/app/shared/ui/badge/src/lib/hlm-badge.ts`
- Create: `frontend/src/app/shared/ui/page-header/page-header.ts`
- Create: `frontend/src/app/shared/ui/page-header/page-header.spec.ts`
- Create: `frontend/src/app/shared/ui/page-header/index.ts`
- Create: `frontend/src/app/shared/ui/data-table/data-table-shell.ts`
- Create: `frontend/src/app/shared/ui/data-table/data-table-shell.spec.ts`
- Create: `frontend/src/app/shared/ui/data-table/index.ts`

**Interfaces:**
- Produces: `<app-page-header>`, `<app-data-table-shell>`, and global `.metric-strip`, `.data-panel`, `.filter-bar`, `.page-stack`, `.mobile-priority` patterns.
- Consumed by: Tasks 5–9.

- [ ] **Step 1: Write failing PageHeader and DataTableShell component tests**

PageHeader test verifies one level-one heading, projected description and projected action button. DataTableShell test verifies a named region and projected real `<table>` remain in the DOM.

- [ ] **Step 2: Run focused tests and verify RED**

Run: `npm test -- --watch=false --include=src/app/shared/ui/page-header/page-header.spec.ts --include=src/app/shared/ui/data-table/data-table-shell.spec.ts`

Expected: FAIL because the components do not exist.

- [ ] **Step 3: Implement minimal standalone components**

`PageHeader` inputs: `eyebrow?: string`, `title: string`, `description?: string`; named content projection for `[page-actions]`. `DataTableShell` input: required `label: string`; host renders a region and projects content without cloning data.

- [ ] **Step 4: Run component tests and verify GREEN**

Run the same focused command; expected all tests pass.

- [ ] **Step 5: Replace decorative global styling with enterprise tokens**

Remove body/page radial gradients and global first-header hero selector. Set radius to 10px base, restrained neutral background, teal primary, flat data panels and shadows only for overlays/sticky surfaces. Preserve reduced-motion behavior.

- [ ] **Step 6: Simplify Spartan primitive visual hierarchy**

Cards use `rounded-xl`, border and no default large shadow. Buttons stop translating vertically on hover. Badges use compact radius and semantic contrast. No behavior/API changes.

- [ ] **Step 7: Verify CSS/template compilation**

Run: `npm run build`

Expected: exit 0 with no Angular template or Tailwind errors. Run `git diff --check`; do not commit.

### Task 5: Persona-specific dashboards

**Files:**
- Modify: `frontend/src/app/features/dashboard/pages/dashboard.ts`
- Modify: `frontend/src/app/features/dashboard/pages/dashboard.html`
- Modify: `frontend/src/app/features/dashboard/pages/dashboard.css`
- Modify: `frontend/src/app/features/dashboard/pages/dashboard.spec.ts`

**Interfaces:**
- Consumes: role/navigation labels and existing Identity, Procurement and Reporting services.
- Produces: role-specific `DashboardWorkspace` with title, mission, primary route/action and metric labels; service calls remain compatible.

- [ ] **Step 1: Add failing table-driven persona tests**

Use literal assertions for Employee, Manager, Finance, Auditor, Admin and AI Operator. Each test checks the primary action and absence of irrelevant technical content. Preserve the real Dashboard component and mock only HTTP-facing services.

Examples:

```ts
expect(renderFor(['employee']).textContent).toContain('Việc cần bạn xử lý');
expect(renderFor(['department_manager']).querySelector('a[href="/approvals"]')).toBeTruthy();
expect(renderFor(['auditor']).textContent).toContain('Chỉ xem');
expect(renderFor(['dx_admin']).querySelector('a[href="/admin"]')).toBeTruthy();
expect(renderFor(['ai_operator']).querySelector('a[href="/ai-center"]')).toBeTruthy();
```

- [ ] **Step 2: Run dashboard tests and verify RED**

Run: `npm test -- --watch=false --include=src/app/features/dashboard/pages/dashboard.spec.ts`

Expected: new persona tests fail because the current shared card dashboard lacks those workspaces/actions.

- [ ] **Step 3: Implement persona workspace configuration**

Create one typed configuration per role for title, mission, primary action, metric labels and read-only context. Keep existing service recovery behavior and avoid new endpoints.

- [ ] **Step 4: Replace generic card grid with context header, metric strip and work queue**

Put role work before secondary analysis. Remove always-visible identity/platform-health cards for business roles; surface an inline error only when a service actually fails.

- [ ] **Step 5: Run dashboard tests and verify GREEN**

Run the focused dashboard command; expected all tests pass.

- [ ] **Step 6: Run App + dashboard integration regression**

Run: `npm test -- --watch=false --include=src/app/app.spec.ts --include=src/app/features/dashboard/pages/dashboard.spec.ts`

Expected: all tests pass. Run build and diff check; do not commit.

### Task 6: Employee and approval workflow screens

**Files:**
- Modify: `frontend/src/app/features/procurement/pages/purchase-request-list/purchase-request-list.html`
- Create: `frontend/src/app/features/procurement/pages/purchase-request-list/purchase-request-list.spec.ts`
- Modify: `frontend/src/app/features/procurement/pages/work-center/work-center.html`
- Modify: `frontend/src/app/features/procurement/pages/work-center/work-center.spec.ts`
- Modify: `frontend/src/app/features/procurement/pages/approval-inbox/approval-inbox.html`
- Modify: `frontend/src/app/features/procurement/pages/approval-inbox/approval-inbox.spec.ts`
- Modify: `frontend/src/app/features/dashboard/pages/notification-center.html`
- Modify: `frontend/src/app/features/dashboard/pages/notification-center.spec.ts`
- Modify: `frontend/src/app/features/dashboard/pages/employee-guide.html`
- Modify: `frontend/src/app/features/dashboard/pages/employee-guide.spec.ts`

**Interfaces:**
- Consumes: PageHeader/DataTableShell and existing component methods/signals.
- Produces: consistent page header, filter bar, priority queue, empty/loading/error states and mobile-primary rows.

**Required state matrix:**
- Purchase Request List: loading, empty, API error, populated success.
- Work Center: loading, empty queue, API error, populated success.
- Approval Inbox: loading, empty queue, API error, populated success.
- Notification Center: loading, empty, API error, unread/read success and disabled bulk action when nothing is selectable.
- Employee Guide is static content and therefore requires successful rendering only, not artificial data states.

- [ ] **Step 1: Add failing workflow semantics tests**

For Purchase Request List, Work Center, Approval Inbox and Notification Center, add one literal test for every state in the required state matrix. Assert accessible queue/region names, recovery copy for API errors, useful empty-state guidance, and the real primary action in populated success. For Notification Center, assert unread/read controls have accessible names and bulk action is disabled with an explanatory description when unavailable. For Employee Guide, assert the four employee jobs link to their real routes.

- [ ] **Step 2: Run the four focused specs and verify RED**

Run: `npm test -- --watch=false --include=src/app/features/procurement/pages/purchase-request-list/purchase-request-list.spec.ts --include=src/app/features/procurement/pages/work-center/work-center.spec.ts --include=src/app/features/procurement/pages/approval-inbox/approval-inbox.spec.ts --include=src/app/features/dashboard/pages/notification-center.spec.ts --include=src/app/features/dashboard/pages/employee-guide.spec.ts`

Expected: at least the new accessible queue/route expectations fail.

- [ ] **Step 3: Refactor templates without changing data methods**

Use page patterns, put work queues before metrics, reduce card count and move secondary details into subordinate text. Every table gets a descriptive caption and `scope="col"`; numeric values are right-aligned.

- [ ] **Step 4: Make primary mobile information visible without horizontal discovery**

At narrow widths show code/title, status, deadline and primary action; secondary columns may hide or move into stacked metadata. Preserve desktop table semantics.

- [ ] **Step 5: Run focused specs and verify GREEN**

Run the same command; expected all focused tests pass.

- [ ] **Step 6: Run procurement/dashboard regression tests and build**

Run: `npm test -- --watch=false`

Then: `npm run build`

Expected: 0 test failures and build exit 0. Review diff; do not commit.

### Task 7: Purchase request create/detail experience

**Files:**
- Modify: `frontend/src/app/features/procurement/pages/purchase-request-create/purchase-request-create.html`
- Modify: `frontend/src/app/features/procurement/pages/purchase-request-create/purchase-request-create.spec.ts`
- Modify: `frontend/src/app/features/procurement/pages/purchase-request-detail/purchase-request-detail.html`
- Create: `frontend/src/app/features/procurement/pages/purchase-request-detail/purchase-request-detail.spec.ts`

**Interfaces:**
- Keeps: all existing form controls, validators, service calls and action methods.
- Produces: step/section hierarchy, sticky draft summary, error summary, sticky role-aware decision panel and readable history/comments.

**Required state matrix:**
- Create/Edit: initial success, invalid submission error summary, save-in-progress disabled actions and service failure feedback.
- Detail: loading, not-found/API error, populated success, action-in-progress disabled state and role-specific read-only state.

- [ ] **Step 1: Add failing create-form hierarchy tests**

Assert one form landmark, the three logical sections, visible draft summary, distinct “Lưu bản nháp” and “Gửi phê duyệt” hierarchy when the state permits, an error summary after invalid submit, disabled actions while saving, and Vietnamese service-failure feedback.

- [ ] **Step 2: Add failing detail action-panel regression tests**

Build literal fixtures for loading, not-found/API error, Employee draft/changes requested, Manager submitted, Finance manager-approved and Auditor read-only. Assert only permitted actions render, in-progress actions become disabled with context, and Auditor receives “Chỉ xem”. Mock the service boundary with complete model fixtures.

- [ ] **Step 3: Run focused specs and verify RED**

Run: `npm test -- --watch=false --include=src/app/features/procurement/pages/purchase-request-create/purchase-request-create.spec.ts --include=src/app/features/procurement/pages/purchase-request-detail/purchase-request-detail.spec.ts`

Expected: tests fail due missing new hierarchy/read-only presentation; existing business assertions remain valid.

- [ ] **Step 4: Refactor create template only**

Reorder existing controls into progressive sections, add sticky summary and semantic error region. Do not alter validators or payload construction.

- [ ] **Step 5: Refactor detail template only**

Use main 2/3 content + sticky action/context rail. Keep comments, attachments, budget and timeline available. Hide no evidence from Auditor; only mutating actions remain governed by existing methods/permissions.

- [ ] **Step 6: Run focused specs and verify GREEN**

Expected: both focused specs pass with clean output.

- [ ] **Step 7: Run full frontend test and build**

Run: `npm test -- --watch=false`

Then: `npm run build`

Expected: 0 failures and build exit 0. Do not commit.

### Task 8: Operations, Finance and supplier screens

**Files:**
- Modify template/spec pairs for `operations-board`, `supplier-directory`, `budget-dashboard`, `invoice-board`, `report-dashboard`.

**Interfaces:**
- Keeps all current service/state/action interfaces.
- Produces queue-first layouts for Finance/Operations and dense but readable reports.

**Required state matrix:**
- Operations: loading, no approved work, API error, populated success and disabled order/receipt actions while submitting.
- Supplier Directory: loading, empty, API error, populated success and disabled save while submitting.
- Budget Dashboard: loading, no allocations, partial/full API error, populated success and disabled adjustment submit while saving.
- Invoice Board: loading, empty queue, API error, populated success and disabled reconciliation/payment actions while saving.
- Report Dashboard: loading, no results, API error, populated success and disabled export/apply action while loading.

- [ ] **Step 1: Add one failing role-job test per screen**

For every screen, add literal tests for each state in the required state matrix. Operations success exposes order creation and delivery stages. Supplier success verifies edit/create hierarchy and directory table. Budget success places alerts and active allocations before history. Invoice success places mismatch/overdue queue before totals. Report success gives filters/results accessible names. Disabled-state tests must assert the user-facing reason or progress label, not only the `disabled` attribute.

- [ ] **Step 2: Run five focused specs and verify RED**

Run each corresponding `*.spec.ts` with `npm test -- --watch=false --include=...`; record the expected behavior failure, not compilation noise.

- [ ] **Step 3: Refactor Operations and Supplier templates**

Use one operational stage strip, one queue table and contextual drawer/card for the active form. Keep order/receipt actions unchanged.

- [ ] **Step 4: Refactor Budget, Invoice and Report templates**

Prioritize exceptions and actionable queues. Convert decorative KPI cards to metric strips; preserve currency grouping and all exported data.

- [ ] **Step 5: Add table caption/scope and mobile column priorities**

Every real table receives semantic headers; primary actions remain discoverable at 320 CSS px without requiring horizontal scroll.

- [ ] **Step 6: Run five focused specs and verify GREEN**

Expected: all focused specs pass.

- [ ] **Step 7: Run full test/build checkpoint**

Run `npm test -- --watch=false`, then `npm run build`, then `git diff --check`. Do not commit.

### Task 9: Audit, policy, admin and AI screens

**Files:**
- Modify: `frontend/src/app/features/reporting/pages/audit-center/audit-center.html`
- Modify: `frontend/src/app/features/reporting/pages/audit-center/audit-center.spec.ts`
- Modify: `frontend/src/app/features/procurement/pages/policy-center/policy-center.html`
- Modify: `frontend/src/app/features/procurement/pages/policy-center/policy-center.spec.ts`
- Modify: `frontend/src/app/features/admin/pages/admin-center/admin-center.html`
- Create: `frontend/src/app/features/admin/pages/admin-center/admin-center.spec.ts`
- Modify: `frontend/src/app/features/ai/pages/recommendation-center/recommendation-center.html`
- Create: `frontend/src/app/features/ai/pages/recommendation-center/recommendation-center.spec.ts`

**Interfaces:**
- Keeps existing services, forms and role guards.
- Produces explicit read-only audit experience, safe policy/admin editing and explainable recommendation review.

**Required state matrix:**
- Audit Center: loading, no evidence/findings, API error, populated read-only success and disabled export while generating.
- Policy Center: loading, empty policy set, API error, populated success and disabled save with reason while saving/invalid.
- Admin Center: loading, no users/departments, API error, populated success and disabled save while saving/invalid.
- Recommendation Center: loading, no recommendations, API error, populated success and disabled decision while submitting.

- [ ] **Step 1: Add failing persona behavior tests**

Create the two missing spec files, then add literal tests for every state in the required state matrix. Audit success checks “Chỉ xem”, evidence filters and immutable log. Policy success checks edit context and reason requirements. Admin success checks user/department task hierarchy. AI success checks confidence/risk/explanation and decision action names. Disabled tests assert progress/reason copy as well as the disabled control.

- [ ] **Step 2: Run four focused specs and verify RED**

Expected: new persona hierarchy/accessibility assertions fail before template changes.

- [ ] **Step 3: Refactor Audit and Policy templates**

Make evidence/findings the first-class content; keep mutation controls limited to existing permissions. Separate policy read state from edit state visually.

- [ ] **Step 4: Refactor Admin and AI templates**

Admin becomes a master-detail management workspace. AI recommendations become review rows/panels with explicit confidence, impact, explanation and auditable decision controls.

- [ ] **Step 5: Run focused specs and verify GREEN**

Expected: all four specs pass.

- [ ] **Step 6: Run complete test/build checkpoint**

Run `npm test -- --watch=false`, `npm run build`, and `git diff --check`. Do not commit.

### Task 10: Accessibility, responsive and content sweep

**Files:**
- Modify: remaining affected templates under `frontend/src/app/**/*.html`
- Modify: `frontend/src/styles.css`
- Modify: existing specs where a real behavior gap is found.
- Modify: `docs/implementation/UI_DESIGN_SYSTEM.md`
- Modify: `docs/USER_GUIDE.md`

**Interfaces:**
- Produces: consistent final language, table semantics, keyboard/focus behavior and documentation matching the new shell.

- [ ] **Step 1: Run automated source inventory**

Run searches for tables without captions/scopes, duplicate h1, remaining `text-4xl`, horizontal mobile navigation, unnecessary English copy and old topbar instructions. Treat findings as an inventory, not proof of accessibility.

Create a state-coverage checklist covering these data pages: Dashboard, Notification Center, Purchase Request List/Create/Detail, Work Center, Approval Inbox, Operations, Suppliers, Budgets, Invoices, Reports, Audit, Policies, Admin and AI Recommendations. For each, record loading/empty/error/success and whether a disabled state applies. A missing state returns ownership to Task 5–9 and must be fixed before continuing.

- [ ] **Step 2: Write failing regression tests for behavioral gaps**

Only add tests where a real user-observable break can be named: missing accessible control name, incorrect route focus, unauthorized action, or dialog not closing. Do not write source-text tests for documentation or CSS.

- [ ] **Step 3: Verify RED for each new regression test**

Run the smallest affected specs and confirm each fails for the intended missing behavior.

- [ ] **Step 4: Fix semantics, focus and responsive edge cases**

Add missing captions/scopes/labels, route focus, dialog focus return and mobile priority rules. Keep tables where tabular comparison is essential; use stacked metadata only when columns are secondary.

- [ ] **Step 5: Update UI design and user guide documentation**

Document sidebar groups, mobile sheet and role-specific dashboards. Remove stale claim that AI Operator lacks an AI menu. Do not regenerate or touch the deleted DOCX.

- [ ] **Step 6: Verify GREEN and build**

Run affected specs, then full `npm test -- --watch=false`, then `npm run build`.

- [ ] **Step 7: Manual keyboard/responsive walkthrough**

At desktop, tablet and 320px-equivalent widths verify: skip link, sidebar toggle, mobile menu Escape/close/focus return, logical tab order, visible focus, primary actions and no hidden critical data. Record limits honestly; no WCAG conformance claim without screen-reader/contrast evidence.

### Task 11: Final review and verification

**Files:**
- Review all uncommitted files except the pre-existing DOCX deletion.

**Interfaces:**
- Consumes: approved spec and Tasks 1–10.
- Produces: evidence-backed completion report; no commit/push.

- [ ] **Step 1: Review requirements line by line**

Map each section of the approved spec to changed files/tests. Record any gap and fix it through a new RED–GREEN cycle before continuing.

- [ ] **Step 2: Run formatter check**

Run: `npm run format:check`

If it fails only on changed frontend files, run the project formatter and re-run the check. Do not format unrelated user files.

- [ ] **Step 3: Run complete test suite fresh**

Run: `npm test -- --watch=false`

Required evidence: exit 0 and 0 failed tests.

- [ ] **Step 4: Run production build fresh**

Run: `npm run build`

Required evidence: exit 0 with no TypeScript/template errors.

- [ ] **Step 5: Run diff hygiene and status checks**

Run: `git diff --check`, `git status --short`, and a scoped diff/stat excluding the pre-existing DOCX deletion. Verify no backend/API/RBAC contract changed.

- [ ] **Step 6: Request final read-only code review**

Provide the approved spec, this plan and the complete uncommitted diff to a reviewer. Fix all Critical/Important findings with TDD and re-run focused verification; document Minor findings.

- [ ] **Step 7: Report verified outcome without committing or pushing**

Report changed areas, exact test/build evidence, manual-review limits and the untouched pre-existing DOCX deletion. Await a separate user request before commit or push.
