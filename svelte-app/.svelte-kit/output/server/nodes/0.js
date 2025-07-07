

export const index = 0;
let component_cache;
export const component = async () => component_cache ??= (await import('../entries/pages/_layout.svelte.js')).default;
export const universal = {
  "prerender": true,
  "ssr": false
};
export const universal_id = "src/routes/+layout.ts";
export const imports = ["_app/immutable/nodes/0.B3FB8zwt.js","_app/immutable/chunks/CWj6FrbW.js","_app/immutable/chunks/DK6efzaC.js","_app/immutable/chunks/D0vgWwoI.js"];
export const stylesheets = ["_app/immutable/assets/0.SvZvJfx9.css"];
export const fonts = [];
