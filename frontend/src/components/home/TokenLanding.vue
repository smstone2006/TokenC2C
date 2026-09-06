<template>
  <div class="token-home" data-testid="token-home">
    <a href="#token-main" class="skip-link">{{
      english ? "Skip to content" : "跳转到正文"
    }}</a>
    <header class="shell masthead">
      <a href="/home" class="wordmark" aria-label="TokenC2C"
        ><img :src="logo" alt="" width="38" height="38" /><span>{{
          brand
        }}</span></a
      >
      <nav
        class="desktop-nav"
        :aria-label="english ? 'Main navigation' : '主导航'"
      >
        <a v-for="(label, i) in copy.nav" :key="label" :href="anchors[i]">{{
          label
        }}</a>
        <router-link v-if="showModelPlaza" to="/model-plaza">{{
          english ? "Models" : "模型广场"
        }}</router-link>
      </nav>
      <div class="nav-actions">
        <LocaleSwitcher /><router-link
          class="login-link"
          :to="authenticated ? '/c2c' : '/login'"
          >{{ authenticated ? copy.workspace : copy.login }}
          <span aria-hidden="true">↗</span></router-link
        >
      </div>
    </header>
    <main id="token-main">
      <section class="shell hero">
        <div class="hero-copy">
          <div class="eyebrow">
            <span class="small-dot"></span>{{ copy.badge }}
          </div>
          <h1>
            {{ copy.title }}<br /><span>{{ copy.accent }}</span>
          </h1>
          <p class="intro">{{ copy.intro }}</p>
          <div class="hero-actions">
            <router-link
              data-testid="buy-cta"
              :to="destination('/keys')"
              class="button primary"
              >{{ copy.buy }} <span aria-hidden="true">↗</span></router-link
            >
            <router-link
              data-testid="sell-cta"
              :to="destination('/c2c')"
              class="button secondary"
              >{{ copy.sell }} <span aria-hidden="true">→</span></router-link
            >
          </div>
          <p class="hero-note">{{ copy.note }}</p>
        </div>
        <figure class="pool-visual" :aria-label="copy.diagram">
          <div class="visual-top">
            <span>TOKEN NETWORK</span><span class="visual-chip">C ↔ C</span>
          </div>
          <div class="network-orbit orbit-one"></div>
          <div class="network-orbit orbit-two"></div>
          <div class="network-line line-one"></div>
          <div class="network-line line-two"></div>
          <div class="network-center">
            <img src="/logo.svg" alt="" width="72" height="72" /><strong>{{
              copy.pool
            }}</strong
            ><span>{{ copy.verified }}</span>
          </div>
          <div class="network-node provider-node">
            <span class="node-symbol" aria-hidden="true">↗</span
            ><strong>{{ copy.supplier }}</strong
            ><small>{{ copy.supplyNote }}</small>
          </div>
          <div class="network-node consumer-node">
            <span class="node-symbol" aria-hidden="true">⌘</span
            ><strong>{{ copy.consumer }}</strong
            ><small>{{ copy.consumeNote }}</small>
          </div>
          <span class="floating-token token-one" aria-hidden="true">T</span
          ><span class="floating-token token-two" aria-hidden="true">T</span>
          <figcaption>{{ copy.diagram }}</figcaption>
        </figure>
      </section>
      <div class="shell provider-strip">
        <span>{{ copy.providers }}</span>
        <div>
          <span v-for="provider in providers" :key="provider">{{
            provider
          }}</span>
        </div>
      </div>
      <section class="rules-band">
        <div class="shell rules">
          <p>{{ copy.ruleLabel }}<span>BUILT FOR BOTH SIDES</span></p>
          <div v-for="(value, i) in ['90%', '10%', '24h']" :key="value">
            <strong>{{ value }}</strong
            ><span>{{ copy.rules[i] }}</span>
          </div>
        </div>
      </section>
      <section id="how-it-works" class="shell section-block">
        <p class="section-kicker">{{ copy.stepsLabel }}</p>
        <h2>{{ copy.stepsTitle }}</h2>
        <div class="steps">
          <article v-for="(step, i) in copy.steps" :key="step[0]">
            <span class="step-number">0{{ i + 1 }}</span>
            <h3>{{ step[0] }}</h3>
            <p>{{ step[1] }}</p>
          </article>
        </div>
      </section>
      <section id="accounting" class="ledger-section">
        <div class="shell ledger-layout">
          <div class="ledger-copy">
            <p class="section-kicker">{{ copy.ledgerLabel }}</p>
            <h2>{{ copy.ledgerTitle }}</h2>
            <p>{{ copy.ledgerIntro }}</p>
            <router-link :to="destination('/c2c')" class="text-link"
              >{{ copy.workspace }}
              <span aria-hidden="true">↗</span></router-link
            >
          </div>
          <div class="ledger-cards">
            <article class="ledger-card buyer-card">
              <div class="card-top">
                <span>01 / CONSUME</span><span aria-hidden="true">↙</span>
              </div>
              <h3>{{ copy.buyerTitle }}</h3>
              <p>{{ copy.buyerDesc }}</p>
              <ul>
                <li v-for="item in copy.buyerItems" :key="item">
                  <span aria-hidden="true">✓</span>{{ item }}
                </li>
              </ul>
            </article>
            <article class="ledger-card seller-card">
              <div class="card-top">
                <span>02 / CONTRIBUTE</span><span aria-hidden="true">↗</span>
              </div>
              <h3>{{ copy.sellerTitle }}</h3>
              <p>{{ copy.sellerDesc }}</p>
              <ul>
                <li v-for="item in copy.sellerItems" :key="item">
                  <span aria-hidden="true">✓</span>{{ item }}
                </li>
              </ul>
            </article>
          </div>
        </div>
      </section>
      <section id="faq" class="shell section-block faq-layout">
        <div>
          <p class="section-kicker">{{ copy.faqLabel }}</p>
          <h2>{{ copy.faqTitle }}</h2>
        </div>
        <div class="faq-items">
          <details v-for="faq in copy.faqs" :key="faq[0]">
            <summary>{{ faq[0] }}<span aria-hidden="true">+</span></summary>
            <p>{{ faq[1] }}</p>
          </details>
        </div>
      </section>
      <section class="shell final-cta">
        <div>
          <p class="section-kicker">LET'S CONNECT.</p>
          <h2>{{ copy.finalTitle }}</h2>
          <p>{{ copy.finalDesc }}</p>
        </div>
        <div class="final-actions">
          <router-link class="button primary" :to="destination('/c2c')"
            >{{ copy.workspace }}
            <span aria-hidden="true">↗</span></router-link
          ><router-link
            v-if="registrationEnabled && !authenticated"
            data-testid="register-cta"
            to="/register"
            class="text-link"
            >{{
              invitationRequired
                ? copy.register
                : english
                  ? "Create an account"
                  : "创建账户"
            }}</router-link
          ><span v-else class="hero-note">{{ copy.closed }}</span>
        </div>
      </section>
    </main>
    <footer class="shell footer">
      <a href="/home" class="wordmark"
        ><img :src="logo" alt="" width="28" height="28" /><span>{{
          brand
        }}</span></a
      >
      <p>{{ copy.footer }}</p>
      <span>© {{ new Date().getFullYear() }} {{ brand }}</span>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import LocaleSwitcher from "@/components/common/LocaleSwitcher.vue";
import { tokenHomeCopy } from "./tokenHomeCopy";

const props = defineProps<{
  brand: string;
  logo: string;
  authenticated: boolean;
  registrationEnabled: boolean;
  invitationRequired: boolean;
  showModelPlaza: boolean;
}>();
const { locale } = useI18n();
const english = computed(() => locale?.value?.startsWith("en") ?? false);
const copy = computed(() => tokenHomeCopy[english.value ? "en" : "zh"]);
const providers = [
  "OpenAI",
  "Anthropic",
  "Gemini",
  "Grok",
  "DeepSeek",
  "Kimi",
  "GLM",
];
const anchors = ["#how-it-works", "#accounting", "#faq"];
function destination(path: string) {
  return props.authenticated
    ? path
    : { path: "/login", query: { redirect: path } };
}
</script>

<style scoped>
.token-home {
  --ink: #102a35;
  --muted: #60747a;
  --mint: #c5f5dc;
  background: #fafcf9;
  color: var(--ink);
  font-family: Inter, "PingFang SC", "Microsoft YaHei", sans-serif;
  -webkit-font-smoothing: antialiased;
}
.shell {
  width: min(1200px, calc(100% - 96px));
  margin: 0 auto;
}
.masthead {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  min-height: 96px;
  border-bottom: 1px solid #dfe6df;
}
.wordmark {
  display: inline-flex;
  align-items: center;
  gap: 11px;
  font-size: 23px;
  font-weight: 750;
  letter-spacing: -1px;
  white-space: nowrap;
}
.desktop-nav,
.nav-actions {
  display: flex;
  align-items: center;
  gap: 28px;
}
.desktop-nav {
  font-size: 13px;
  color: var(--muted);
}
.nav-actions {
  gap: 18px;
}
.login-link {
  font-size: 13px;
  font-weight: 650;
  white-space: nowrap;
}
.login-link span {
  margin-left: 12px;
}
.token-home a:hover {
  color: #16805d;
}
.token-home a:focus-visible,
.token-home summary:focus-visible {
  outline: 3px solid #16805d;
  outline-offset: 5px;
}
.skip-link {
  position: absolute;
  left: 20px;
  top: -100px;
  background: #fff;
  padding: 12px;
  z-index: 50;
}
.skip-link:focus {
  top: 12px;
}
.hero {
  display: grid;
  grid-template-columns: 1.16fr 1fr;
  gap: 55px;
  align-items: center;
  padding: 80px 0 68px;
}
.eyebrow {
  display: flex;
  gap: 9px;
  align-items: center;
  font-size: 12px;
  letter-spacing: 1px;
  margin-bottom: 26px;
  color: #436658;
}
.small-dot {
  width: 7px;
  height: 7px;
  background: #42886c;
  border-radius: 50%;
}
h1 {
  font-size: clamp(38px, 4.1vw, 61px);
  font-weight: 750;
  letter-spacing: -2.8px;
  line-height: 1.24;
  margin: 0 0 26px;
}
h1 > span {
  color: #2b8766;
}
.intro {
  font-size: 16px;
  line-height: 1.95;
  color: var(--muted);
  max-width: 510px;
}
.hero-actions {
  display: flex;
  gap: 12px;
  margin-top: 32px;
}
.button {
  display: inline-flex;
  gap: 25px;
  align-items: center;
  justify-content: center;
  padding: 16px 23px;
  border-radius: 7px;
  font-size: 14px;
  font-weight: 650;
  transition: background 0.2s;
  min-height: 52px;
}
.primary {
  background: var(--ink);
  color: white;
}
.token-home .primary:hover {
  background: #244653;
  color: white;
}
.secondary {
  background: transparent;
  border: 1px solid #bbc9c2;
}
.hero-note {
  font-size: 11px;
  color: var(--muted);
  margin-top: 17px;
  line-height: 1.7;
}
.pool-visual {
  height: 410px;
  position: relative;
  margin: 0;
  border: 1px solid #dce7de;
  border-radius: 16px;
  background:
    radial-gradient(ellipse at center, #e0f4e7, transparent 68%), #f0f5ee;
  overflow: hidden;
}
.visual-top {
  position: absolute;
  top: 24px;
  left: 24px;
  right: 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 10px;
  font-weight: 650;
  letter-spacing: 2px;
  color: #638176;
}
.visual-chip {
  letter-spacing: 0;
  border: 1px solid #c4d7c9;
  border-radius: 5px;
  padding: 4px 8px;
}
.network-orbit {
  position: absolute;
  left: 50%;
  top: 50%;
  border: 1px solid #cbddd0;
  border-radius: 50%;
  transform: translate(-50%, -50%);
  width: 228px;
  height: 228px;
}
.orbit-two {
  width: 345px;
  height: 345px;
}
.network-line {
  position: absolute;
  top: 48%;
  height: 1px;
  background: #a6c5b4;
  width: 43%;
  transform: rotate(32deg);
  left: 6%;
}
.line-two {
  left: 50%;
  top: 54%;
}
.network-center {
  position: absolute;
  left: 50%;
  top: 47%;
  transform: translate(-50%, -50%);
  display: flex;
  align-items: center;
  flex-direction: column;
  gap: 10px;
  z-index: 2;
}
.network-center img {
  border-radius: 18px;
  box-shadow: 0 16px 32px #25503920;
}
.network-center strong {
  font-size: 14px;
}
.network-center > span {
  font-size: 10px;
  color: #618270;
  background: #e5f1e7;
  padding: 4px 9px;
  border-radius: 20px;
}
.network-node {
  position: absolute;
  background: #fff;
  border: 1px solid #d6e0d7;
  border-radius: 9px;
  box-shadow: 0 10px 20px #243b2b06;
  padding: 14px;
  z-index: 3;
  width: 159px;
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.provider-node {
  left: 17px;
  top: 86px;
}
.consumer-node {
  right: 17px;
  bottom: 67px;
}
.network-node strong {
  font-size: 12px;
}
.network-node small {
  font-size: 9px;
  color: var(--muted);
}
.node-symbol {
  color: #307c59;
  font-size: 18px;
  margin-bottom: 2px;
}
.floating-token {
  position: absolute;
  border: 1px solid #cbddd0;
  border-radius: 50%;
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  font-size: 11px;
  background: #eff9f0;
  color: #668a6f;
}
.token-one {
  right: 60px;
  top: 81px;
}
.token-two {
  left: 80px;
  bottom: 88px;
}
figcaption {
  position: absolute;
  bottom: 20px;
  left: 0;
  right: 0;
  text-align: center;
  color: #758c7d;
  font-size: 10px;
}
.provider-strip {
  display: flex;
  align-items: center;
  gap: 38px;
  border-top: 1px solid #dfe6df;
  padding: 25px 0 31px;
}
.provider-strip > span {
  font-size: 11px;
  color: var(--muted);
  white-space: nowrap;
}
.provider-strip > div {
  display: flex;
  flex: 1;
  justify-content: space-between;
  gap: 15px;
  flex-wrap: wrap;
}
.provider-strip > div > span {
  font-size: 17px;
  font-weight: 650;
  letter-spacing: -0.5px;
  color: #4f645c;
}
.rules-band {
  background: var(--ink);
  color: #fff;
}
.rules {
  display: grid;
  grid-template-columns: 1.5fr 1fr 1fr 1fr;
  padding: 35px 0;
  gap: 20px;
  align-items: center;
}
.rules > p {
  font-size: 15px;
}
.rules > p > span {
  display: block;
  color: #92b1b2;
  font-size: 9px;
  letter-spacing: 1.6px;
  margin-top: 9px;
}
.rules > div {
  border-left: 1px solid #35505a;
  padding-left: 38px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.rules strong {
  font-size: 36px;
  line-height: 1.2;
  letter-spacing: -1px;
  font-weight: 550;
  color: var(--mint);
}
.rules > div > span {
  font-size: 11px;
  color: #b8c9cb;
}
.section-block {
  padding: 85px 0;
}
.section-kicker {
  font-size: 10px;
  letter-spacing: 2px;
  color: #577768;
  margin-bottom: 17px;
  font-weight: 600;
}
h2 {
  font-size: 32px;
  line-height: 1.4;
  letter-spacing: -1px;
  font-weight: 650;
  white-space: pre-line;
}
.steps {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 50px;
  margin-top: 40px;
}
.step-number {
  display: block;
  font-size: 12px;
  color: #549774;
  border-bottom: 1px solid #d8e2d9;
  padding-bottom: 18px;
  margin-bottom: 23px;
  font-family: monospace;
}
.steps h3 {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 12px;
}
.steps p {
  font-size: 13px;
  line-height: 1.9;
  color: var(--muted);
}
.ledger-section {
  background: #eef3ee;
  border-top: 1px solid #e1e7df;
  border-bottom: 1px solid #e1e7df;
}
.ledger-layout {
  padding: 75px 0;
  display: grid;
  grid-template-columns: 0.88fr 1.2fr;
  gap: 55px;
  align-items: center;
}
.ledger-copy > p:not(.section-kicker) {
  font-size: 13px;
  line-height: 1.9;
  color: var(--muted);
  margin: 20px 0;
}
.text-link {
  font-size: 12px;
  font-weight: 650;
  display: inline-flex;
  gap: 28px;
}
.ledger-cards {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 15px;
}
.ledger-card {
  padding: 25px;
  border: 1px solid #d6e2d7;
  border-radius: 10px;
  background: #fafcf9;
}
.seller-card {
  background: #dceee2;
  margin-top: 27px;
  transform: translateY(0);
}
.buyer-card {
  margin-bottom: 27px;
}
.card-top {
  display: flex;
  justify-content: space-between;
  font-size: 9px;
  letter-spacing: 1px;
  color: #60776b;
  margin-bottom: 28px;
}
.card-top > span:last-child {
  font-size: 22px;
}
.ledger-card h3 {
  font-size: 20px;
  font-weight: 650;
}
.ledger-card > p {
  font-size: 11px;
  color: var(--muted);
  margin-top: 6px;
}
.ledger-card ul {
  margin-top: 23px;
  padding-top: 18px;
  border-top: 1px solid #d0ddd2;
  list-style: none;
}
.ledger-card li {
  display: flex;
  gap: 9px;
  font-size: 11px;
  line-height: 1.65;
  margin-top: 12px;
}
.ledger-card li > span {
  color: #448469;
}
.faq-layout {
  display: grid;
  grid-template-columns: 1fr 1.5fr;
  gap: 60px;
}
.faq-items details {
  border-bottom: 1px solid #dfe6df;
}
.faq-items details:first-child {
  border-top: 1px solid #dfe6df;
}
.faq-items summary {
  cursor: pointer;
  list-style: none;
  display: flex;
  justify-content: space-between;
  gap: 16px;
  font-size: 14px;
  font-weight: 550;
  padding: 22px 0;
}
.faq-items summary::-webkit-details-marker {
  display: none;
}
.faq-items summary > span {
  font-size: 20px;
  font-weight: 400;
}
.faq-items details[open] summary > span {
  transform: rotate(45deg);
}
.faq-items details > p {
  font-size: 13px;
  color: var(--muted);
  line-height: 1.9;
  padding: 0 28px 22px 0;
}
.final-cta {
  padding: 40px 45px;
  background: #e0f1e4;
  border: 1px solid #cfe1d2;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 30px;
  margin-bottom: 65px;
}
.final-cta h2 {
  font-size: 28px;
}
.final-cta p:not(.section-kicker) {
  font-size: 13px;
  color: var(--muted);
  margin-top: 12px;
}
.final-actions {
  display: flex;
  flex-direction: column;
  gap: 16px;
  align-items: center;
  flex-shrink: 0;
}
.final-actions .hero-note {
  margin: 0;
  font-size: 10px;
}
.footer {
  padding: 26px 0 32px;
  border-top: 1px solid #dfe6df;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
}
.footer .wordmark {
  font-size: 17px;
  letter-spacing: -0.5px;
}
.footer > p,
.footer > span {
  font-size: 10px;
  color: var(--muted);
}
@media (min-width: 1500px) {
  .hero {
    padding-top: 95px;
    padding-bottom: 85px;
  }
  .pool-visual {
    height: 440px;
  }
}
@media (max-width: 1100px) {
  .shell {
    width: calc(100% - 56px);
  }
  .desktop-nav {
    gap: 18px;
  }
  .hero {
    gap: 24px;
  }
  h1 {
    font-size: 40px;
    letter-spacing: -1.8px;
  }
  .hero-actions {
    flex-wrap: wrap;
  }
  .network-node {
    width: 135px;
    padding: 11px;
  }
  .provider-node {
    left: 12px;
  }
  .consumer-node {
    right: 12px;
  }
  .ledger-layout {
    grid-template-columns: 1fr;
    gap: 30px;
  }
  .ledger-cards {
    max-width: 660px;
  }
  .provider-strip {
    align-items: flex-start;
  }
  .rules > div {
    padding-left: 24px;
  }
}
@media (max-width: 760px) {
  .shell {
    width: calc(100% - 40px);
  }
  .masthead {
    min-height: 78px;
    gap: 12px;
  }
  .desktop-nav {
    display: none;
  }
  .wordmark {
    font-size: 21px;
    gap: 8px;
  }
  .nav-actions {
    gap: 9px;
  }
  .login-link {
    font-size: 12px;
  }
  .login-link span {
    margin-left: 3px;
  }
  .hero {
    grid-template-columns: 1fr;
    padding: 44px 0 30px;
    gap: 30px;
  }
  h1 {
    font-size: clamp(32px, 8vw, 49px);
    letter-spacing: -1.5px;
  }
  .eyebrow {
    font-size: 10px;
  }
  .intro {
    font-size: 14px;
  }
  .hero-actions {
    gap: 10px;
    margin-top: 24px;
  }
  .button {
    padding: 14px 18px;
    font-size: 13px;
    gap: 18px;
  }
  .hero-note {
    font-size: 10px;
  }
  .pool-visual {
    height: 365px;
  }
  .provider-strip {
    flex-direction: column;
    gap: 17px;
    padding: 24px 0;
  }
  .provider-strip > div {
    justify-content: flex-start;
    column-gap: 23px;
    row-gap: 14px;
  }
  .provider-strip > div > span {
    font-size: 15px;
  }
  .rules {
    grid-template-columns: repeat(3, 1fr);
    gap: 24px 10px;
    padding: 28px 0;
  }
  .rules > p {
    grid-column: 1 / -1;
  }
  .rules > div {
    padding-left: 15px;
  }
  .rules strong {
    font-size: 29px;
  }
  .rules > div > span {
    font-size: 10px;
  }
  .section-block {
    padding: 52px 0;
  }
  h2 {
    font-size: 27px;
  }
  .steps {
    grid-template-columns: 1fr;
    gap: 30px;
    margin-top: 25px;
  }
  .step-number {
    padding-bottom: 12px;
    margin-bottom: 15px;
  }
  .steps h3 {
    font-size: 17px;
  }
  .ledger-layout {
    padding: 45px 0;
  }
  .ledger-cards {
    gap: 10px;
  }
  .ledger-card {
    padding: 17px 13px;
  }
  .ledger-card h3 {
    font-size: 18px;
  }
  .ledger-card li {
    font-size: 10px;
  }
  .card-top {
    font-size: 8px;
    letter-spacing: 0.5px;
  }
  .faq-layout {
    grid-template-columns: 1fr;
    gap: 25px;
  }
  .faq-items summary {
    font-size: 13px;
  }
  .final-cta {
    padding: 28px 22px;
    flex-direction: column;
    align-items: flex-start;
    margin-bottom: 35px;
  }
  .final-cta h2 {
    font-size: 25px;
  }
  .final-actions {
    align-items: flex-start;
  }
  .footer {
    flex-wrap: wrap;
    gap: 15px;
  }
  .footer > p {
    width: 100%;
    order: 3;
  }
  .section-kicker {
    font-size: 9px;
  }
}
</style>
