// GraphQL documents. Fields requested here must exist in graph/schema.graphqls.

const POSITION = `x y`;
const FRAME = `tape pc`;

const STEP_EVENT_FIELDS = `
  step
  instruction
  tapeIndex
  slotIndex
  from { ${POSITION} }
  to { ${POSITION} }
  facingBefore
  facingAfter
  blocked
  collected { ${POSITION} }
  callStack { ${FRAME} }
  status
  lossReason
`;

const MAP_FIELDS = `
  id
  name
  description
  difficulty
  width
  height
  layout
  start { ${POSITION} }
  startFacing
  rocks { ${POSITION} }
  treats { ${POSITION} }
  mainTapeLength
  subTapeLengths
  maxSteps
  maxCallDepth
  allowRecursion
`;

/** Full session snapshot: everything the board, tapes and counters need. */
export const SESSION_FIELDS = `
  id
  displayName
  mapId
  map { ${MAP_FIELDS} }
  program { main subs }
  vm {
    pos { ${POSITION} }
    facing
    treatsRemaining { ${POSITION} }
    callStack { ${FRAME} }
    steps
    status
    lossReason
    visited { ${POSITION} }
  }
  playing
  speedMs
  seq
  createdAt
  lastActionAt
  attempts
  lastEvent { ${STEP_EVENT_FIELDS} }
`;

export const MAPS_QUERY = `
  query Maps {
    maps { ${MAP_FIELDS} }
  }
`;

export const MAP_QUERY = `
  query MapById($id: ID!) {
    map(id: $id) { ${MAP_FIELDS} }
  }
`;

export const SESSION_QUERY = `
  query Session($id: ID!) {
    session(id: $id) { ${SESSION_FIELDS} }
  }
`;

export const SESSIONS_QUERY = `
  query Sessions($limit: Int, $sort: SessionSort, $mapId: ID) {
    sessions(sort: $sort, limit: $limit, mapId: $mapId) {
      id
      displayName
      mapId
      playing
      createdAt
      lastActionAt
      map { name }
      vm { status steps treatsRemaining { ${POSITION} } }
    }
  }
`;

export const SIMULATE_QUERY = `
  query Simulate($mapID: ID!, $program: ProgramInput!, $includeEvents: Boolean) {
    simulate(mapID: $mapID, program: $program, includeEvents: $includeEvents) {
      status
      lossReason
      steps
    }
  }
`;

export const CREATE_SESSION_MUTATION = `
  mutation CreateSession($mapID: ID!, $displayName: String) {
    createSession(mapID: $mapID, displayName: $displayName) { ${SESSION_FIELDS} }
  }
`;

export const DELETE_SESSION_MUTATION = `
  mutation DeleteSession($id: ID!) {
    deleteSession(id: $id)
  }
`;

export const RENAME_SESSION_MUTATION = `
  mutation RenameSession($id: ID!, $displayName: String!) {
    renameSession(id: $id, displayName: $displayName) { ${SESSION_FIELDS} }
  }
`;

export const SET_PROGRAM_MUTATION = `
  mutation SetProgram($sessionID: ID!, $program: ProgramInput!) {
    setProgram(sessionID: $sessionID, program: $program) { ${SESSION_FIELDS} }
  }
`;

export const RUN_MUTATION = `
  mutation Run($sessionID: ID!, $speedMs: Int) {
    run(sessionID: $sessionID, speedMs: $speedMs) { ${SESSION_FIELDS} }
  }
`;

export const PAUSE_MUTATION = `
  mutation Pause($sessionID: ID!) {
    pause(sessionID: $sessionID) { ${SESSION_FIELDS} }
  }
`;

export const STEP_MUTATION = `
  mutation Step($sessionID: ID!) {
    step(sessionID: $sessionID) { ${SESSION_FIELDS} }
  }
`;

export const RESET_MUTATION = `
  mutation Reset($sessionID: ID!) {
    reset(sessionID: $sessionID) { ${SESSION_FIELDS} }
  }
`;

export const SESSION_UPDATED_SUBSCRIPTION = `
  subscription SessionUpdated($sessionID: ID!) {
    sessionUpdated(sessionID: $sessionID) {
      seq
      session { ${SESSION_FIELDS} }
      event { ${STEP_EVENT_FIELDS} }
    }
  }
`;

export const CREATE_MAP_MUTATION = `
  mutation CreateMap($map: MapInput!) {
    createMap(map: $map) { ${MAP_FIELDS} }
  }
`;

export const UPDATE_MAP_MUTATION = `
  mutation UpdateMap($map: MapInput!) {
    updateMap(map: $map) { ${MAP_FIELDS} }
  }
`;

export const DELETE_MAP_MUTATION = `
  mutation DeleteMap($id: ID!) {
    deleteMap(id: $id)
  }
`;

export const VALIDATE_MAP_MUTATION = `
  mutation ValidateMap($map: MapInput!) {
    validateMap(map: $map) {
      valid
      issues { message }
      unreachableTreats { ${POSITION} }
    }
  }
`;
