import { beforeEach, describe, expect, it, vi } from "vitest";
import { mount, RouterLinkStub } from "@vue/test-utils";
import TokenLanding from "@/components/home/TokenLanding.vue";

const { locale } = vi.hoisted(() => ({ locale: { value: "zh" } }));
vi.mock("vue-i18n", async (importOriginal) => ({
  ...(await importOriginal<typeof import("vue-i18n")>()),
  useI18n: () => ({ locale }),
}));

function render(overrides = {}) {
  return mount(TokenLanding, {
    props: {
      brand: "TokenC2C",
      logo: "/logo.svg",
      authenticated: false,
      registrationEnabled: true,
      invitationRequired: true,
      showModelPlaza: false,
      ...overrides,
    },
    global: { stubs: { RouterLink: RouterLinkStub, LocaleSwitcher: true } },
  });
}
function target(wrapper: ReturnType<typeof render>, testId: string) {
  return wrapper
    .findAllComponents(RouterLinkStub)
    .find((link) => link.attributes("data-testid") === testId)
    ?.props("to");
}
describe("TokenC2C landing page", () => {
  beforeEach(() => {
    locale.value = "zh";
  });
  it("explains both roles and the actual credit settlement terms", () => {
    const wrapper = render();
    expect(wrapper.text()).toContain("作为使用者");
    expect(wrapper.text()).toContain("作为供应者");
    expect(wrapper.text()).toContain("90%");
    expect(wrapper.text()).toContain("不提供现金提现");
    expect(wrapper.get("figcaption").text()).toContain("非实时交易数据");
    expect(wrapper.findAll("h1")).toHaveLength(1);
  });
  it("preserves the intended buyer/seller destination through sign-in", () => {
    const wrapper = render();
    expect(target(wrapper, "buy-cta")).toEqual({
      path: "/login",
      query: { redirect: "/keys" },
    });
    expect(target(wrapper, "sell-cta")).toEqual({
      path: "/login",
      query: { redirect: "/c2c" },
    });
  });
  it("takes signed-in members directly to their tools", () => {
    const wrapper = render({ authenticated: true });
    expect(target(wrapper, "buy-cta")).toBe("/keys");
    expect(target(wrapper, "sell-cta")).toBe("/c2c");
    expect(wrapper.find('[data-testid="register-cta"]').exists()).toBe(false);
  });
  it("respects disabled registration and invitation requirements", () => {
    expect(
      render({ registrationEnabled: false })
        .find('[data-testid="register-cta"]')
        .exists(),
    ).toBe(false);
    expect(render().get('[data-testid="register-cta"]').text()).toBe(
      "凭邀请码加入",
    );
    expect(
      render({ invitationRequired: false })
        .get('[data-testid="register-cta"]')
        .text(),
    ).toBe("创建账户");
  });
  it("offers English content and working section anchors", () => {
    locale.value = "en";
    const wrapper = render();
    expect(wrapper.get("h1").text()).toContain("Every Token.");
    for (const anchor of ["how-it-works", "accounting", "faq"])
      expect(wrapper.find(`#${anchor}`).exists()).toBe(true);
    expect(wrapper.get('[data-testid="register-cta"]').text()).toBe(
      "Join with an invite",
    );
  });
});
