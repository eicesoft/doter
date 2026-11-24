import type { Snapshot } from "../../bindings/doter/internal/notice/models";

/**
 * 前端 mock 通知数据，用于在没有「完全磁盘访问」权限时测试通知面板。
 * 把 USE_MOCK_NOTICES 设为 true 即可在 useNotices 中启用。
 *
 * 数据覆盖：单条、同应用多条堆叠、不同时间粒度、title/body 缺省组合。
 */
export const USE_MOCK_NOTICES = false;

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/** 每次调用都按当前时间生成，保证「刚刚 / 分钟前 / 昨天」始终合理。 */
export function mockNoticesSnapshot(): Snapshot {
  const now = Date.now();
  return {
    error: "",
    updatedAt: now,
    new: [],
    entries: [
      // 同一应用多条 —— 测试堆叠、数量角标、展开/收起。
      {
        id: "mock-mail-1",
        app: "com.apple.mail",
        title: "新邮件",
        body: "张伟：Q3 季度报告已发送，请查收附件",
        date: now - 2 * MINUTE,
      },
      {
        id: "mock-mail-2",
        app: "com.apple.mail",
        title: "送达提醒",
        body: "你寄往 client@example.com 的邮件已被阅读",
        date: now - 3 * HOUR,
      },
      {
        id: "mock-mail-3",
        app: "com.apple.mail",
        title: "收件箱",
        body: "你有 5 封未读邮件",
        date: now - DAY,
      },
      // 同一应用两条 —— 第二条堆叠层。
      {
        id: "mock-wechat-1",
        app: "com.tencent.xinWeChat",
        title: "李雷",
        body: "明天下午 3 点开会，记得准备一下",
        date: now - 12 * MINUTE,
      },
      {
        id: "mock-wechat-2",
        app: "com.tencent.xinWeChat",
        title: "项目群",
        body: "王芳：设计稿已经更新到最新版",
        date: now - 5 * HOUR,
      },
      {
        id: "mock-lark-1",
        app: "com.bytedance.lark",
        title: "日历提醒",
        body: "产品评审会将在 10 分钟后开始",
        date: now - 40 * MINUTE,
      },
      // 只有 title —— 测试 body 缺省。
      {
        id: "mock-sys-1",
        app: "com.apple.systempreferences",
        title: "软件更新可用",
        body: "",
        date: now - 2 * DAY,
      },
      // 只有 body —— 测试 title 缺省。
      {
        id: "mock-calendar-1",
        app: "com.apple.calendar",
        title: "",
        body: "团队周会现在开始",
        date: now - 30_000,
      },
    ],
  };
}
