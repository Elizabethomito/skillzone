import initSqlJs from "sql.js";
import wasmUrl from "sql.js/dist/sql-wasm.wasm?url";

let db = null;
let dbInitPromise = null;

export async function initDatabase() {
  if (db) return db;
  if (dbInitPromise) return dbInitPromise;

  dbInitPromise = (async () => {
    let SQL;
    try {
      SQL = await initSqlJs({
        locateFile: (file) => wasmUrl,
      });
    } catch (e) {
      console.error('Failed to initialize sql.js WASM:', e);
      throw e;
    }

    const savedData = localStorage.getItem("skillzone_db");

    if (savedData) {
      try {
        const buf = new Uint8Array(JSON.parse(savedData));
        db = new SQL.Database(buf);
      } catch {
        db = new SQL.Database();
      }
    } else {
      db = new SQL.Database();
    }

    try {
      const info = db.exec("SELECT sql FROM sqlite_master WHERE type='table' AND name='users'");
      console.debug('users table schema after init:', info);
    } catch (e) {
      console.debug('Could not read users table schema after init', e);
    }

    db.run(`
      CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        role TEXT NOT NULL CHECK(role IN ('user', 'organization')),
        first_name TEXT,
        last_name TEXT,
        organization_name TEXT,
        email TEXT UNIQUE NOT NULL,
        password_hash TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
      )
    `);

    db.run(`
      CREATE TABLE IF NOT EXISTS events (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        description TEXT,
        organization_id INTEGER NOT NULL,
        skill_category TEXT,
        status TEXT DEFAULT 'open' CHECK(status IN ('open', 'ongoing', 'completed')),
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY(organization_id) REFERENCES users(id)
      )
    `);

    db.run(`
      CREATE TABLE IF NOT EXISTS registrations (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        event_id INTEGER NOT NULL,
        status TEXT DEFAULT 'applied' CHECK(status IN ('applied', 'confirmed', 'completed')),
        verified INTEGER DEFAULT 0,
        FOREIGN KEY(user_id) REFERENCES users(id),
        FOREIGN KEY(event_id) REFERENCES events(id)
      )
    `);

    db.run(`
      CREATE TABLE IF NOT EXISTS badges (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        event_id INTEGER NOT NULL,
        skill_name TEXT NOT NULL,
        issued_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY(user_id) REFERENCES users(id),
        FOREIGN KEY(event_id) REFERENCES events(id)
      )
    `);

    db.run(`
      CREATE TABLE IF NOT EXISTS links (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        org_id INTEGER NOT NULL,
        user_id INTEGER NOT NULL,
        employer_name TEXT NOT NULL,
        employer_contact TEXT,
        note TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY(org_id) REFERENCES users(id),
        FOREIGN KEY(user_id) REFERENCES users(id)
      )
    `);

    persist();
    console.debug('Database initialized. No seeding performed.');
    return db;
  })();

  return dbInitPromise;
}

function ensureDb() {
  if (!db) throw new Error('Database not initialized. Call initDatabase() first.');
  return db;
}

export function getDb() {
  return db;
}

export function persist() {
  if (db) {
    const data = db.export();
    localStorage.setItem('skillzone_db', JSON.stringify(Array.from(data)));
  }
}

function rowsToObjects(result) {
  if (!result || result.length === 0) return [];
  const { columns, values } = result[0];
  return values.map((row) => {
    const obj = {};
    columns.forEach((col, i) => (obj[col] = row[i]));
    return obj;
  });
}

function queryRows(sql, params) {
  const d = ensureDb();
  const stmt = d.prepare(sql);
  try {
    if (params) {
      if (Array.isArray(params)) stmt.bind(params);
      else stmt.bind(params);
    }
    const rows = [];
    while (stmt.step()) {
      rows.push(stmt.getAsObject());
    }
    return rows;
  } finally {
    try {
      stmt.free();
    } catch (e) {
      console.debug('Failed to free stmt in queryRows', e);
    }
  }
}

export async function hashPassword(password) {
  const encoder = new TextEncoder();
  const data = encoder.encode(password);
  const hash = await crypto.subtle.digest('SHA-256', data);
  return Array.from(new Uint8Array(hash))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

// ── User operations ──

export async function createUser({ role, firstName, lastName, organizationName, email, password }) {
  const d = ensureDb();
  if (!role || (role !== 'user' && role !== 'organization')) {
    throw new Error('Invalid or missing role: must be "user" or "organization"');
  }
  if (!email) throw new Error('Email is required');
  if (!password) throw new Error('Password is required');

  const passwordHash = await hashPassword(password);
  const params = {
    $role: role,
    $first_name: firstName || null,
    $last_name: lastName || null,
    $organization_name: organizationName || null,
    $email: email,
    $password_hash: passwordHash,
  };
    try {
      console.debug('createUser params (no password):', { ...params, $password_hash: '***' });
      try {
        const tableSql = d.exec("SELECT sql FROM sqlite_master WHERE type='table' AND name='users'");
        console.debug('users table schema before insert:', tableSql);
      } catch (e) {
        console.debug('Could not read users table schema before insert', e);
      }
      // Use a prepared statement with named parameters to ensure proper binding
      const stmt = d.prepare(
        `INSERT INTO users (role, first_name, last_name, organization_name, email, password_hash) VALUES ($role, $first_name, $last_name, $organization_name, $email, $password_hash)`
      );
      try {
        stmt.bind(params);
        stmt.step();
      } finally {
        try {
          stmt.free();
        } catch (e) {
          console.debug('Failed to free statement', e);
        }
      }
    persist();
    const resultRows = queryRows(`SELECT * FROM users WHERE email = $email`, { $email: email });
    return resultRows[0] || null;
  } catch (e) {
    console.error('createUser failed', { role, firstName, lastName, organizationName, email }, e);
    if (e.message && e.message.includes('UNIQUE')) throw new Error('Email already exists');
    throw e;
  }
}

export async function authenticateUser(email, password) {
  const d = ensureDb();
  const passwordHash = await hashPassword(password);
  const rows = queryRows(`SELECT * FROM users WHERE email = $email AND password_hash = $ph`, {
    $email: email,
    $ph: passwordHash,
  });
  return rows[0] || null;
}

export function getUserById(id) {
  const d = ensureDb();
  const rows = queryRows(`SELECT * FROM users WHERE id = $id`, { $id: id });
  return rows[0] || null;
}

export function updateUser(id, fields) {
  const d = ensureDb();
  const sets = [];
  const vals = [];
  Object.entries(fields).forEach(([key, val]) => {
    sets.push(`${key} = ?`);
    vals.push(val);
  });
  vals.push(id);
  d.run(`UPDATE users SET ${sets.join(', ')} WHERE id = ?`, vals);
  persist();
  return getUserById(id);
}

export function deleteUser(id) {
  const d = ensureDb();
  d.run(`DELETE FROM badges WHERE user_id = ?`, [id]);
  d.run(`DELETE FROM registrations WHERE user_id = ?`, [id]);
  d.run(`DELETE FROM events WHERE organization_id = ?`, [id]);
  d.run(`DELETE FROM users WHERE id = ?`, [id]);
  persist();
}

// ── Event operations ──

export function createEvent({ title, description, organizationId, skillCategory }) {
  const d = ensureDb();
  if (!title || !title.toString().trim()) throw new Error('Title is required');
  if (!organizationId) throw new Error('Organization ID is required');
  const titleVal = title?.toString?.().trim();
  const descVal = description ? description.toString().trim() : null;
  const catVal = skillCategory ? skillCategory.toString().trim() : null;
  console.debug('createEvent will insert with values:', { titleVal, descVal, organizationId, catVal });
  try {
    const stmt = d.prepare(
      `INSERT INTO events (title, description, organization_id, skill_category) VALUES ($t, $d, $org, $cat)`
    );
    try {
      const bindObj = { $t: titleVal, $d: descVal, $org: organizationId, $cat: catVal };
      console.debug('createEvent binding object types:', {
        $t: typeof bindObj.$t,
        $d: bindObj.$d === null ? 'null' : typeof bindObj.$d,
        $org: typeof bindObj.$org,
        $cat: bindObj.$cat === null ? 'null' : typeof bindObj.$cat,
      });
      stmt.bind(bindObj);
      stmt.step();
    } finally {
      try {
        stmt.free();
      } catch (e) {
        console.debug('Failed to free createEvent stmt', e);
      }
    }
  } catch (e) {
    try {
      console.error('createEvent params', { title: title?.toString?.(), description: description?.toString?.(), organizationId, skillCategory });
      const schema = d.exec("SELECT sql FROM sqlite_master WHERE type='table' AND name='events'");
      console.error('events table schema:', schema);
      const info = d.exec("PRAGMA table_info('events')");
      console.error('events table info:', info);
      const recent = d.exec(`SELECT * FROM events ORDER BY id DESC LIMIT 5`);
      console.error('recent events rows:', recent);
      const count = d.exec(`SELECT COUNT(*) as count FROM events`);
      console.error('events count:', count);
    } catch (inner) {
      console.error('createEvent debug inner failed', inner);
    }
    console.error('createEvent run failed', e);
    throw e;
  }
  persist();
  try {
    const recent = queryRows(`SELECT e.*, u.organization_name FROM events e JOIN users u ON e.organization_id = u.id ORDER BY e.id DESC LIMIT 1`);
    const created = recent[0] || null;
    console.info('createEvent created event row (recent):', created);
    return created;
  } catch (e) {
    console.error('createEvent post-insert read failed', e);
    return null;
  }
}

export function getEventById(id) {
  const d = ensureDb();
  const rows = queryRows(
    `SELECT e.*, u.organization_name FROM events e JOIN users u ON e.organization_id = u.id WHERE e.id = $id`,
    { $id: id }
  );
  return rows[0] || null;
}

export function getAllEvents() {
  const d = ensureDb();
  const rows = queryRows(`SELECT e.*, u.organization_name FROM events e JOIN users u ON e.organization_id = u.id ORDER BY e.created_at DESC`);
  try {
    console.debug('getAllEvents returned', rows.length, 'rows');
  } catch (e) {
    console.debug('getAllEvents debug failed', e);
  }
  return rows;
}

export function getEventsByOrganization(orgId) {
  const d = ensureDb();
  const rows = queryRows(
    `SELECT e.*, u.organization_name FROM events e JOIN users u ON e.organization_id = u.id WHERE e.organization_id = $org ORDER BY e.created_at DESC`,
    { $org: orgId }
  );
  return rows;
}

export function updateEventStatus(eventId, status) {
  const d = ensureDb();
  d.run(`UPDATE events SET status = ? WHERE id = ?`, [status, eventId]);
  persist();
}

// ── Registration operations ──

export function registerForEvent(userId, eventId) {
  const d = ensureDb();
  console.info('registerForEvent called', { userId, eventId });
  const existing = queryRows(`SELECT * FROM registrations WHERE user_id = $u AND event_id = $e`, {
    $u: userId,
    $e: eventId,
  });
  if (existing.length > 0) throw new Error('Already registered');
  try {
    console.debug('registerForEvent types', { userIdType: typeof userId, eventIdType: typeof eventId });
    console.debug('registerForEvent about to prepare INSERT');
    const stmt = d.prepare(`INSERT INTO registrations (user_id, event_id) VALUES ($u, $e)`);
    console.debug('registerForEvent prepared successfully, now binding...');
    try {
      const bindObj = { $u: userId, $e: eventId };
      console.debug('registerForEvent bindObj', bindObj);
      stmt.bind(bindObj);
      console.debug('registerForEvent bound successfully, now stepping...');
      stmt.step();
      console.debug('registerForEvent step succeeded');
    } finally {
      try { stmt.free(); } catch (e) { console.debug('free register stmt failed', e); }
    }
    persist();
    const created = queryRows(`SELECT * FROM registrations ORDER BY id DESC LIMIT 1`)[0] || null;
    console.info('registerForEvent succeeded', { userId, eventId, created });
    return created;
  } catch (e) {
    try {
      console.error('registerForEvent failed to insert', { userId, eventId, typeofUserId: typeof userId, typeofEventId: typeof eventId });
      const schema = d.exec("SELECT sql FROM sqlite_master WHERE type='table' AND name='registrations'");
      console.error('registrations table schema:', schema);
      const info = d.exec("PRAGMA table_info('registrations')");
      console.error('registrations table info:', info);
      const recent = d.exec(`SELECT * FROM registrations ORDER BY id DESC LIMIT 5`);
      console.error('recent registrations rows:', recent);
    } catch (inner) {
      console.error('registerForEvent debug inner failed', inner);
    }
    throw e;
  }
}

export function unregisterFromEvent(userId, eventId) {
  const d = ensureDb();
  console.info('unregisterFromEvent called', { userId, eventId });
  try {
    const stmt = d.prepare(`DELETE FROM registrations WHERE user_id = $u AND event_id = $e`);
    try {
      stmt.bind({ $u: userId, $e: eventId });
      stmt.step();
      console.info('unregisterFromEvent DELETE executed');
    } finally {
      try { stmt.free(); } catch (e) { console.debug('free stmt failed', e); }
    }
    persist();
    const remaining = queryRows(`SELECT COUNT(*) as count FROM registrations WHERE user_id = $u AND event_id = $e`, { $u: userId, $e: eventId });
    console.info('unregisterFromEvent succeeded, remaining registrations:', remaining[0]?.count || 0);
  } catch (e) {
    console.error('unregisterFromEvent failed', e, { userId, eventId });
    throw e;
  }
}

export function removeUserFromEvent(userId, eventId) {
  const d = ensureDb();
  console.info('removeUserFromEvent called (org side)', { userId, eventId });
  try {
    const stmt = d.prepare(`DELETE FROM registrations WHERE user_id = $u AND event_id = $e`);
    try {
      stmt.bind({ $u: userId, $e: eventId });
      stmt.step();
      console.info('removeUserFromEvent DELETE executed');
    } finally {
      try { stmt.free(); } catch (e) { console.debug('free stmt failed', e); }
    }
    persist();
    const remaining = queryRows(`SELECT COUNT(*) as count FROM registrations WHERE user_id = $u AND event_id = $e`, { $u: userId, $e: eventId });
    console.info('removeUserFromEvent succeeded, remaining registrations:', remaining[0]?.count || 0);
  } catch (e) {
    console.error('removeUserFromEvent failed', e, { userId, eventId });
    throw e;
  }
}

export function getUserRegistrations(userId) {
  const d = ensureDb();
  const rows = queryRows(
    `SELECT r.*, e.title, e.skill_category, e.status as event_status, u.organization_name
     FROM registrations r
     JOIN events e ON r.event_id = e.id
     JOIN users u ON e.organization_id = u.id
     WHERE r.user_id = $uid`,
    { $uid: userId }
  );
  return rows;
}

export function getEventApplicants(eventId) {
  const d = ensureDb();
  const rows = queryRows(
    `SELECT r.*, u.first_name, u.last_name, u.email
     FROM registrations r
     JOIN users u ON r.user_id = u.id
     WHERE r.event_id = $eid`,
    { $eid: eventId }
  );
  return rows;
}

export function updateRegistrationStatus(regId, status) {
  const d = ensureDb();
  d.run(`UPDATE registrations SET status = ? WHERE id = ?`, [status, regId]);
  persist();
}

export function confirmParticipation(regId) {
  const d = ensureDb();
  console.info('confirmParticipation called', { regId });
  try {
    const stmt = d.prepare(`UPDATE registrations SET status = $s, verified = $v WHERE id = $id`);
    try {
      stmt.bind({ $s: 'confirmed', $v: 1, $id: regId });
      stmt.step();
      console.info('confirmParticipation UPDATE executed');
    } finally {
      try { stmt.free(); } catch (e) { console.debug('free stmt failed', e); }
    }
    persist();
    const updated = queryRows(`SELECT * FROM registrations WHERE id = $id`, { $id: regId });
    console.info('confirmParticipation succeeded', updated[0]);
  } catch (e) {
    console.error('confirmParticipation failed', e, { regId });
    throw e;
  }
}

export function completeRegistration(regId) {
  const d = ensureDb();
  console.info('completeRegistration called', { regId });
  try {
    const stmt = d.prepare(`UPDATE registrations SET status = $s WHERE id = $id`);
    try {
      stmt.bind({ $s: 'completed', $id: regId });
      stmt.step();
      console.info('completeRegistration UPDATE executed');
    } finally {
      try { stmt.free(); } catch (e) { console.debug('free stmt failed', e); }
    }
    persist();
    const updated = queryRows(`SELECT * FROM registrations WHERE id = $id`, { $id: regId });
    console.info('completeRegistration succeeded', updated[0]);
  } catch (e) {
    console.error('completeRegistration failed', e, { regId });
    throw e;
  }
}

// ── Badge operations ──

export function issueBadge(userId, eventId, skillName) {
  const d = ensureDb();
  const existing = queryRows(`SELECT * FROM badges WHERE user_id = $u AND event_id = $e`, {
    $u: userId,
    $e: eventId,
  });
  if (existing.length > 0) return null;
  d.run(`INSERT INTO badges (user_id, event_id, skill_name) VALUES (?, ?, ?)`, [userId, eventId, skillName]);
  persist();
}

export function getUserBadges(userId) {
  const d = ensureDb();
  const rows = queryRows(
    `SELECT b.*, e.title as event_title
     FROM badges b
     JOIN events e ON b.event_id = e.id
     WHERE b.user_id = $uid
     ORDER BY b.issued_at DESC`,
    { $uid: userId }
  );
  return rows;
}

/**
 * Get badge counts grouped by skill category for a user.
 * Returns array of { skill_name, count }.
 */
export function getUserBadgeCounts(userId) {
  const d = ensureDb();
  const rows = queryRows(
    `SELECT skill_name, COUNT(*) as count
     FROM badges WHERE user_id = $uid
     GROUP BY skill_name
     ORDER BY count DESC`,
    { $uid: userId }
  );
  return rows;
}

/**
 * Get all applicants across all events for an organization, with their badge counts and progress.
 */
export function getOrgApplicantsWithProgress(orgId) {
  const d = ensureDb();
  const rows = queryRows(
    `SELECT u.id as user_id, u.first_name, u.last_name, u.email,
            COUNT(DISTINCT r.event_id) as total_registrations,
            SUM(CASE WHEN r.status = 'completed' THEN 1 ELSE 0 END) as completed_events,
            SUM(CASE WHEN r.status = 'confirmed' THEN 1 ELSE 0 END) as ongoing_events,
            (SELECT COUNT(*) FROM badges b WHERE b.user_id = u.id) as total_badges
     FROM registrations r
     JOIN users u ON r.user_id = u.id
     JOIN events e ON r.event_id = e.id
     WHERE e.organization_id = $org
     GROUP BY u.id
     ORDER BY completed_events DESC`,
    { $org: orgId }
  );
  return rows;
}

export function getOrgStats(orgId) {
  const d = ensureDb();
  const events = queryRows(`SELECT COUNT(*) as count FROM events WHERE organization_id = $org`, { $org: orgId });
  const active = queryRows(`SELECT COUNT(*) as count FROM events WHERE organization_id = $org AND status != 'completed'`, { $org: orgId });
  const applicants = queryRows(
    `SELECT COUNT(*) as count FROM registrations r JOIN events e ON r.event_id = e.id WHERE e.organization_id = $org`,
    { $org: orgId }
  );
  return {
    totalEvents: events[0]?.count || 0,
    activeEvents: active[0]?.count || 0,
    totalApplicants: applicants[0]?.count || 0,
  };
}

// Links (organization -> talent -> employer)
export function createLink(orgId, userId, employerName, employerContact = null, note = null) {
  const d = ensureDb();
  const stmt = d.prepare(`INSERT INTO links (org_id, user_id, employer_name, employer_contact, note) VALUES ($org, $user, $emp, $contact, $note)`);
  try {
    stmt.bind({ $org: orgId, $user: userId, $emp: employerName, $contact: employerContact, $note: note });
    stmt.step();
  } finally {
    try { stmt.free(); } catch (e) { console.debug('free createLink stmt failed', e); }
  }
  persist();
}

export function getLinksForOrg(orgId) {
  return queryRows(`SELECT l.*, u.first_name, u.last_name, u.email FROM links l JOIN users u ON l.user_id = u.id WHERE l.org_id = $org ORDER BY l.created_at DESC`, { $org: orgId });
}

export function getLinksForUser(userId) {
  return queryRows(`SELECT l.*, ou.organization_name as org_name FROM links l JOIN users ou ON l.org_id = ou.id WHERE l.user_id = $u ORDER BY l.created_at DESC`, { $u: userId });
}
