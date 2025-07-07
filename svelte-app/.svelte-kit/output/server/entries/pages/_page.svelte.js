import { y as head, v as pop, t as push } from "../../chunks/index.js";
function _page($$payload, $$props) {
  push();
  head($$payload, ($$payload2) => {
    $$payload2.title = `<title>Xanthus - Infrastructure Management</title>`;
  });
  $$payload.out += `<div class="container mx-auto px-4 py-8"><div class="text-center"><h1 class="text-4xl font-bold text-gray-900 mb-4">Welcome to Xanthus Svelte</h1> <p class="text-xl text-gray-600 mb-8">Modern infrastructure management with Svelte frontend</p> `;
  {
    $$payload.out += "<!--[!-->";
  }
  $$payload.out += `<!--]--></div></div>`;
  pop();
}
export {
  _page as default
};
