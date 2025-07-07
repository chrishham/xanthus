import"../chunks/CWj6FrbW.js";import"../chunks/RaIwcHWQ.js";import{o as X}from"../chunks/ChR5l_r1.js";import{p as E,m as V,t as I,aB as N,n as B,o as M,w as g,y as o,A as R,x as v,E as O,u as l,at as y,aa as z,l as F,k as G,g as H}from"../chunks/Ce1HVTUQ.js";import{c as m,s as f,a as K}from"../chunks/bgO99CsW.js";import{i as D}from"../chunks/BdY6-5Q-.js";import{p as q,s as U,a as J}from"../chunks/C18abfhG.js";import{_ as $}from"../chunks/CmsKOCeN.js";import{a as Q,e as i}from"../chunks/BJkf_VXN.js";import{L as Y}from"../chunks/DLRiGjVN.js";import{s as Z}from"../chunks/DSBPkOg0.js";const tt=!0,et=tt,at=!0,st=!1,wt=Object.freeze(Object.defineProperty({__proto__:null,prerender:at,ssr:st},Symbol.toStringTag,{value:"Module"}));var rt=V('<nav class="bg-white shadow-md border-b"><div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"><div class="flex justify-between h-16"><div class="flex items-center"><img src="/static/icons/logo.png" alt="Xanthus Logo" class="w-8 h-8 mr-3"/> <h1 class="text-xl font-semibold text-gray-900">Xanthus</h1></div> <div class="flex items-center space-x-4"><a href="/main">Dashboard</a> <a href="/dns">DNS Config</a> <a href="/vps">VPS Management</a> <a href="/applications">Applications</a> <a href="/version">Version</a> <button class="text-gray-600 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium">About</button> <a href="/logout" class="text-red-600 hover:text-red-800 px-3 py-2 rounded-md text-sm font-medium">Logout</a></div></div></div></nav>');function ot(t,a){E(a,!1);let _=q(a,"currentPage",8,"");async function w(){try{const e=await Q.get("/about");if(et){const{default:d}=await $(async()=>{const{default:p}=await import("../chunks/0MHz4Was.js");return{default:p}},[],import.meta.url);d.fire({title:"About Xanthus",html:`
						<div class="text-left space-y-4">
							<div class="text-center mb-4">
								<img src="/static/icons/logo.png" alt="Xanthus Logo" class="w-16 h-16 mx-auto mb-2">
								<h3 class="text-xl font-bold text-gray-900">Xanthus</h3>
								<p class="text-gray-600">Configuration-Driven Infrastructure Management Platform</p>
							</div>
							
							<div class="grid grid-cols-2 gap-4 text-sm">
								<div>
									<span class="font-semibold text-gray-700">Version:</span>
									<span class="text-gray-900">${e.version}</span>
								</div>
								<div>
									<span class="font-semibold text-gray-700">Build Date:</span>
									<span class="text-gray-900">${e.build_date}</span>
								</div>
								<div>
									<span class="font-semibold text-gray-700">Go Version:</span>
									<span class="text-gray-900">${e.go_version}</span>
								</div>
								<div>
									<span class="font-semibold text-gray-700">Platform:</span>
									<span class="text-gray-900">${e.platform}</span>
								</div>
							</div>
							
							<div class="border-t pt-4">
								<h4 class="font-semibold text-gray-700 mb-2">Features</h4>
								<ul class="text-sm text-gray-600 space-y-1">
									<li>• VPS provisioning (Hetzner Cloud & Oracle Cloud)</li>
									<li>• DNS & SSL management via Cloudflare</li>
									<li>• K3s Kubernetes orchestration</li>
									<li>• Configuration-driven application deployment</li>
									<li>• Self-updating platform capabilities</li>
								</ul>
							</div>
							
							<div class="border-t pt-4 text-center">
								<p class="text-xs text-gray-500">
									Open source project licensed under MIT<br>
									<a href="https://github.com/your-org/xanthus" target="_blank" class="text-blue-600 hover:text-blue-800">View on GitHub</a>
								</p>
							</div>
						</div>
					`,width:500,showCancelButton:!1,confirmButtonText:"Close",confirmButtonColor:"#6B7280"})}}catch(e){console.error("Error showing about modal:",e);{const{default:d}=await $(async()=>{const{default:p}=await import("../chunks/0MHz4Was.js");return{default:p}},[],import.meta.url);d.fire("Error","Failed to load about information: "+e.message,"error")}}}function s(e){return _()===e}function r(e){if(s(e))switch(e){case"dns":return"text-blue-600 bg-blue-50 px-3 py-2 rounded-md text-sm font-medium";case"vps":return"text-blue-600 bg-blue-50 px-3 py-2 rounded-md text-sm font-medium";case"applications":return"text-purple-600 bg-purple-50 px-3 py-2 rounded-md text-sm font-medium";case"version":return"text-green-600 bg-green-50 px-3 py-2 rounded-md text-sm font-medium";case"main":default:return"text-gray-900 bg-gray-50 px-3 py-2 rounded-md text-sm font-medium"}return"text-gray-600 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"}D();var n=rt(),x=g(n),c=g(x),h=o(g(c),2),S=g(h),A=o(S,2),P=o(A,2),C=o(P,2),L=o(C,2),j=o(L,2);R(2),v(h),v(c),v(x),v(n),I((e,d,p,T,W)=>{f(S,1,e),f(A,1,d),f(P,1,p),f(C,1,T),f(L,1,W)},[()=>m(l(()=>r("main"))),()=>m(l(()=>r("dns"))),()=>m(l(()=>r("vps"))),()=>m(l(()=>r("applications"))),()=>m(l(()=>r("version")))],O),N("click",j,w),B(t,n),M()}const nt=()=>{const t=Z;return{page:{subscribe:t.page.subscribe},navigating:{subscribe:t.navigating.subscribe},updated:t.updated}},it={subscribe(t){return nt().page.subscribe(t)}},lt={user:null,loading:!1,error:null,isAuthenticated:!1},u=z(lt);y(u,t=>t.user);y(u,t=>t.isAuthenticated);y(u,t=>t.loading);y(u,t=>t.error);const b=t=>{u.update(a=>({...a,user:t,isAuthenticated:!!t,error:null}))},k=t=>{u.update(a=>({...a,loading:t}))},ut=async()=>{k(!0);try{const t=await fetch("/auth/status");if(t.ok){const a=await t.json();a.authenticated?b(a.user||{id:"1",username:"user",isAuthenticated:!0}):b(null)}else b(null)}catch(t){console.error("Auth check error:",t),b(null)}finally{k(!1)}};var ct=V('<div class="min-h-screen bg-gray-100"><!> <!> <!></div>');function St(t,a){E(a,!1);const[_,w]=U(),s=()=>J(it,"$page",_);X(()=>{ut()}),F(()=>(s(),i),()=>{s().url.pathname==="/app"?i("main"):s().url.pathname.startsWith("/app/applications")?i("applications"):s().url.pathname.startsWith("/app/vps")?i("vps"):s().url.pathname.startsWith("/app/dns")?i("dns"):s().url.pathname.startsWith("/app/version")&&i("version")}),G(),D();var r=ct(),n=g(r);const x=O(()=>(s(),l(()=>s().url.pathname.includes("/app/")&&s().url.pathname.split("/")[2]||"main")));ot(n,{get currentPage(){return H(x)}});var c=o(n,2);K(c,a,"default",{});var h=o(c,2);Y(h,{}),v(r),B(t,r),M(),w()}export{St as component,wt as universal};
