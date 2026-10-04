var e={NOT_FOUND:`Not found`,INVALID_PROGRAM:`Invalid program`,SESSION_PLAYING:`Session is running — pause it first`,SESSION_TERMINAL:`Run is over — reset to try again`,INVALID_ARGUMENT:`Invalid input`,FORBIDDEN:`Forbidden — check the admin key`};function t(t){let n=t.find(e=>e&&(e.message||e.extensions?.code));if(!n)return null;let r=typeof n.extensions?.code==`string`?n.extensions.code:null,i=n.message??``;return{code:r,title:r&&e[r]||`Request failed`,message:i}}function n(e){if(Array.isArray(e)){let n=t(e);if(n)return n}else if(e&&typeof e==`object`){let n=e;if(n.graphQLErrors?.length){let e=t(n.graphQLErrors);if(e)return e}if(n.networkError)return{code:null,title:`Cannot reach the server`,message:n.networkError.message??``};if(n.message)return{code:null,title:`Request failed`,message:n.message}}return{code:null,title:`Request failed`,message:typeof e==`string`?e:``}}function r(e){switch(e){case`STEP_LIMIT`:return`The robot ran out of steps before collecting every treat.`;case`CALL_DEPTH`:return`Subroutine calls nested too deep (call depth limit exceeded).`;case`PROGRAM_ENDED`:return`The program finished with treats still left on the board.`;default:return`The run ended without collecting every treat.`}}var i=`x y`,a=`tape pc`,o=`
  step
  instruction
  tapeIndex
  slotIndex
  from { ${i} }
  to { ${i} }
  facingBefore
  facingAfter
  blocked
  collected { ${i} }
  callStack { ${a} }
  status
  lossReason
`,s=`
  id
  name
  description
  difficulty
  width
  height
  layout
  start { ${i} }
  startFacing
  rocks { ${i} }
  treats { ${i} }
  mainTapeLength
  subTapeLengths
  maxSteps
  maxCallDepth
  allowRecursion
`,c=`
  id
  displayName
  mapId
  map { ${s} }
  program { main subs }
  vm {
    pos { ${i} }
    facing
    treatsRemaining { ${i} }
    callStack { ${a} }
    steps
    status
    lossReason
    visited { ${i} }
  }
  playing
  speedMs
  seq
  createdAt
  lastActionAt
  attempts
  lastEvent { ${o} }
`,l=`
  query Maps {
    maps { ${s} }
  }
`,u=`
  query MapById($id: ID!) {
    map(id: $id) { ${s} }
  }
`,d=`
  query Session($id: ID!) {
    session(id: $id) { ${c} }
  }
`,f=`
  query Sessions($limit: Int, $sort: SessionSort, $mapId: ID) {
    sessions(sort: $sort, limit: $limit, mapId: $mapId) {
      id
      displayName
      mapId
      playing
      createdAt
      lastActionAt
      map { name }
      vm { status steps treatsRemaining { ${i} } }
    }
  }
`,p=`
  mutation CreateSession($mapID: ID!, $displayName: String) {
    createSession(mapID: $mapID, displayName: $displayName) { ${c} }
  }
`;`${c}`;var m=`
  mutation SetProgram($sessionID: ID!, $program: ProgramInput!) {
    setProgram(sessionID: $sessionID, program: $program) { ${c} }
  }
`,h=`
  mutation Run($sessionID: ID!, $speedMs: Int) {
    run(sessionID: $sessionID, speedMs: $speedMs) { ${c} }
  }
`,g=`
  mutation Pause($sessionID: ID!) {
    pause(sessionID: $sessionID) { ${c} }
  }
`,_=`
  mutation Step($sessionID: ID!) {
    step(sessionID: $sessionID) { ${c} }
  }
`,v=`
  mutation Reset($sessionID: ID!) {
    reset(sessionID: $sessionID) { ${c} }
  }
`,y=`
  subscription SessionUpdated($sessionID: ID!) {
    sessionUpdated(sessionID: $sessionID) {
      seq
      session { ${c} }
      event { ${o} }
    }
  }
`,b=`
  mutation CreateMap($map: MapInput!) {
    createMap(map: $map) { ${s} }
  }
`,x=`
  mutation UpdateMap($map: MapInput!) {
    updateMap(map: $map) { ${s} }
  }
`,S=`
  mutation DeleteMap($id: ID!) {
    deleteMap(id: $id)
  }
`,C=`
  mutation ValidateMap($map: MapInput!) {
    validateMap(map: $map) {
      valid
      issues { message }
      unreachableTreats { ${i} }
    }
  }
`;export{r as _,u as a,h as c,y as d,m as f,n as g,C as h,l as i,f as l,x as m,p as n,g as o,_ as p,S as r,v as s,b as t,d as u};