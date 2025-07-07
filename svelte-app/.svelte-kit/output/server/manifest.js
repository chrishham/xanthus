export const manifest = (() => {
function __memo(fn) {
	let value;
	return () => value ??= (value = fn());
}

return {
	appDir: "_app",
	appPath: "app/_app",
	assets: new Set(["favicon.ico"]),
	mimeTypes: {},
	_: {
		client: {start:"_app/immutable/entry/start.C7bNGzWG.js",app:"_app/immutable/entry/app.D1qhwZYs.js",imports:["_app/immutable/entry/start.C7bNGzWG.js","_app/immutable/chunks/DhvT9-Cm.js","_app/immutable/chunks/BKCjr91X.js","_app/immutable/chunks/D0vgWwoI.js","_app/immutable/entry/app.D1qhwZYs.js","_app/immutable/chunks/D0vgWwoI.js","_app/immutable/chunks/BKCjr91X.js","_app/immutable/chunks/CWj6FrbW.js","_app/immutable/chunks/CtuYc89S.js"],stylesheets:[],fonts:[],uses_env_dynamic_public:false},
		nodes: [
			__memo(() => import('./nodes/0.js')),
			__memo(() => import('./nodes/1.js'))
		],
		routes: [
			
		],
		prerendered_routes: new Set(["/app/"]),
		matchers: async () => {
			
			return {  };
		},
		server_assets: {}
	}
}
})();
