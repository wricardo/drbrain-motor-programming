import{A as e,I as t,L as n,N as r,Q as i,S as a,W as o,Y as s,Z as c,_ as l,a as u,d,et as f,j as p,k as m,lt as h,rt as g,tt as _,u as v,z as y}from"./CZgXN-iS.js";import"./xihTtKlq.js";import{r as b}from"./CX2fdOv1.js";var x=r(`<p role="alert" class="mt-2 text-xs text-amber-800">Could not access the clipboard — the prompt is selected below, press Ctrl/Cmd+C.</p>`),S=r(`<div class="rounded-xl border border-indigo-200 bg-white p-4" data-testid="ai-prompt-card"><div class="flex items-start justify-between gap-3"><div class="min-w-0"><h2 class="text-xs font-semibold uppercase tracking-widest text-indigo-900"> </h2> <p class="mt-1 text-sm text-slate-700"> </p></div> <button type="button" data-testid="ai-prompt-copy"> </button></div> <p class="sr-only" role="status" aria-live="polite"> </p> <!> <details class="mt-3"><summary class="cursor-pointer select-none text-xs font-medium text-slate-700 hover:text-slate-900">Show full prompt</summary> <textarea readonly="" rows="14" aria-label="Prompt for an AI" class="mt-2 w-full resize-y rounded-lg border border-indigo-100 bg-slate-50 p-3 font-mono text-xs leading-relaxed text-slate-700"></textarea></details></div>`);function C(t,r){let b=u(r,`title`,3,`Play with an AI`),C=_(!1),w=_(!1),T=_(!1),E=_(null),D;async function O(){f(w,!1);try{await navigator.clipboard.writeText(r.prompt),f(C,!0),clearTimeout(D),D=setTimeout(()=>f(C,!1),2e3)}catch{f(w,!0),f(T,!0),queueMicrotask(()=>y(E)?.select())}}var k=S(),A=s(k),j=s(A),M=s(j),N=c(M,!0),P=i(M,2),F=c(P,!0);h(j);var I=i(j,2),L=c(I,!0);h(A);var R=i(A,2),z=c(R,!0),B=i(R,2),V=e=>{var t=x();p(e,t)};m(B,e=>{y(w)&&e(V)});var H=i(B,2),U=i(s(H),2);g(U),d(U,e=>f(E,e),()=>y(E)),h(H),h(k),o(()=>{e(N,b()),e(F,r.blurb),a(I,1,`shrink-0 rounded-full border px-4 py-2 text-sm font-semibold transition-colors ${y(C)?`border-emerald-300 bg-emerald-50 text-emerald-800`:`border-indigo-300 text-indigo-800 hover:bg-indigo-50`}`),e(L,y(C)?`Copied!`:`Copy prompt`),e(z,y(C)?`Prompt copied to clipboard`:``),l(U,r.prompt)}),n(`click`,I,O),n(`click`,U,e=>e.currentTarget.select()),v(`open`,`toggle`,H,e=>f(T,e),()=>y(T)),p(t,k)}t([`click`]);var w={EMPTY:{label:`Empty`,glyph:``,key:``,classes:`bg-white text-slate-400 border-slate-300`},TURN_LEFT:{label:`Turn left`,glyph:`↺`,key:`L`,classes:`bg-sky-100 text-sky-900 border-sky-400`},TURN_RIGHT:{label:`Turn right`,glyph:`↻`,key:`R`,classes:`bg-amber-100 text-amber-900 border-amber-400`},MOVE_FORWARD:{label:`Move forward`,glyph:`↑`,key:`F`,classes:`bg-emerald-100 text-emerald-900 border-emerald-400`},CALL_SUB_1:{label:`Call sub 1`,glyph:`S1`,key:`1`,classes:`bg-violet-100 text-violet-900 border-violet-400`},CALL_SUB_2:{label:`Call sub 2`,glyph:`S2`,key:`2`,classes:`bg-fuchsia-100 text-fuchsia-900 border-fuchsia-400`},CALL_SUB_3:{label:`Call sub 3`,glyph:`S3`,key:`3`,classes:`bg-rose-100 text-rose-900 border-rose-400`}};function T(e){return w[e]}var E=[`CALL_SUB_1`,`CALL_SUB_2`,`CALL_SUB_3`];function D(e){let t=E.indexOf(e);return t<0?null:t}function O(e,t){let n=D(e);return n===null||n<t}function k(e){return[`MOVE_FORWARD`,`TURN_LEFT`,`TURN_RIGHT`,...E.slice(0,Math.max(0,Math.min(e,E.length)))]}function A(e,t){let n=e.toUpperCase();for(let e of Object.keys(w))if(w[e].key===n&&O(e,t))return e;return null}function j(e){return e<0?`Main`:`Sub ${e+1}`}var M=e=>[`EMPTY`,`TURN_LEFT`,`TURN_RIGHT`,`MOVE_FORWARD`,...Array.from({length:e},(e,t)=>`CALL_SUB_${t+1}`)].join(`, `),N=e=>`[${Array(e).fill(`EMPTY`).join(`, `)}]`;function P(e){let t=e.reduce((e,t)=>Math.max(e,t.length),0),n=String(Math.max(0,e.length-1)).length;return[` `.repeat(n+3)+Array.from({length:t},(e,t)=>String(t%10)).join(``),...e.map((e,t)=>`y${String(t).padStart(n,` `)}  ${e}`)].join(`
`)}var F={UP:`up (y-1)`,RIGHT:`right (x+1)`,DOWN:`down (y+1)`,LEFT:`left (x-1)`};function I(e,t){let n=t.map,r=n.subTapeLengths,i=[`main = ${n.mainTapeLength} slots`,...r.map((e,t)=>`sub${t+1} (subs[${t}]) = ${e} slots`)].join(`; `),a=`main: ${N(n.mainTapeLength)}\n    subs: [${r.map(N).join(`, `)}]`,o=n.treats.map(e=>`(${e.x},${e.y})`).join(` `),s=n.allowRecursion?`Recursion is ALLOWED on this map: a sub may call itself (or any sub already running). The call stack is bounded by maxCallDepth = ${n.maxCallDepth} and by the step limit, and there is no tail-call optimisation (the stack stays deep until a sub finishes).`:`Recursion is NOT allowed on this map: a CALL_SUB_k aimed at the sub that is running, or at any sub already waiting on the call stack, does nothing (the step is still spent). A sub may call a different sub that is not on the stack, so you can nest subs; maxCallDepth = ${n.maxCallDepth}.`;return`Use this GraphQL API to play an existing ${b} session and WIN it.

${b} is a robot programming puzzle. You do not steer the robot: you write it a program (a main tape plus ${r.length} subroutine ${r.length===1?`tape`:`tapes`} of instructions), the server runs it one instruction per step, and you win the moment every treat has been collected.

Session ID: ${t.id}
GraphQL endpoint: ${e}/graphql
Playground: ${e}/playground
Full rules and every field: ${e}/llms.txt
Spectators can watch live at: ${e}/watch/${t.id}

GraphQL introspection is usually enabled; use the Playground or __schema/__type if you need to discover fields.

## This map: ${n.name} (${n.width}x${n.height}, ${n.difficulty})
${n.description}

Layout (x = column, y = row; row 0 is the TOP and y grows DOWNWARD):
${P(n.layout)}

'.' empty, '#' rock (blocks), '*' treat, '^ > v <' = robot start cell and initial facing.
Robot starts at (${n.start.x},${n.start.y}) facing ${F[n.startFacing]??n.startFacing}.
Treats (${n.treats.length}): ${o}

Tapes (setProgram needs EXACTLY these lengths; pad unused slots with EMPTY): ${i}.
Limits: at most ${n.maxSteps} steps; ${s}

## Rules that matter
- Instructions: ${M(r.length)}.
- Each executed instruction is one step (a blocked move and a CALL both count). EMPTY slots are skipped for free.
- MOVE_FORWARD into a rock or off the map is a harmless no-op (still a step, "blocked: true"). Treats are collected on arrival.
- When a sub tape ends, control returns to the caller; when the main tape ends with treats left, you lose (PROGRAM_ENDED). Other losses: STEP_LIMIT, CALL_DEPTH.
- Win beats the step limit on the same step. setProgram and reset put the robot back at the start. setProgram is rejected while the session is playing (SESSION_PLAYING); finished sessions refuse run/step until reset (SESSION_TERMINAL).

## How to win
1. Plan on paper first: write the route as straight runs and turns using the coordinates above. Tapes are tiny, so look for repeated patterns and put them in subroutines (a sub can be a "row", a "U-turn", a "spiral side"...). Remember a blocked move costs a step but does no harm, so an over-long run of MOVE_FORWARD is safe when a wall will stop it.
2. Store the program in the session:
mutation {
  setProgram(sessionID: "${t.id}", program: {
    ${a}
  }) { id vm { status steps } }
}
   (Replace the EMPTY slots with real instructions.)
3. Run it on the server (50-5000 ms per step) or step manually, then read the result:
mutation { run(sessionID: "${t.id}", speedMs: 100) { playing vm { status steps } } }
mutation { step(sessionID: "${t.id}") { vm { status steps pos { x y } } lastEvent { instruction blocked } } }
query { session(id: "${t.id}") { playing attempts bestSteps vm { status lossReason steps pos { x y } facing treatsRemaining { x y } callStack { tape pc } } } }
4. If the status is LOST, read lossReason and treatsRemaining, fix the program, and repeat from step 2 (call reset first if you want to run again without changing the program). Do not stop until vm.status is WON, then report the winning program and its step count.

Other mutations: pause, reset, renameSession, deleteSession. Do not try to guess other sessions' ids and do not delete this one.`}function L(e){return`Play ${b}, a robot programming puzzle, through its GraphQL API and win it.

You write the robot a program (a main tape plus up to 3 subroutine tapes); the server runs it one instruction per step; you win when every treat has been collected. Tapes are tiny, so the trick is spotting repeated patterns and reusing them with subroutines.

1. Read ${e}/llms.txt first: it has the complete rules, the coordinate convention (row 0 is the top, y grows downward) and working examples for every step.
2. GraphQL endpoint: ${e}/graphql (Playground: ${e}/playground).
3. List the maps: query { maps { id name difficulty layout mainTapeLength subTapeLengths maxSteps maxCallDepth allowRecursion } } and pick one (start with an easy one).
4. Create your own session: mutation { createSession(mapID: "MAP_ID", displayName: "my agent") { id } }
5. Plan the route from the layout, then setProgram, run, and read the session until vm.status is WON. If it is LOST, read lossReason, fix the program and try again.
6. Tell me the session id so I can watch it live at ${e}/watch/SESSION_ID, and report the winning program and its step count.`}export{k as a,T as i,I as n,j as o,A as r,C as s,L as t};