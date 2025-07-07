import { w as slot } from "../../chunks/index.js";
function _layout($$payload, $$props) {
  $$payload.out += `<main class="min-h-screen bg-gray-50"><!---->`;
  slot($$payload, $$props, "default", {});
  $$payload.out += `<!----></main>`;
}
export {
  _layout as default
};
