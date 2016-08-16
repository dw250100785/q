package q

import (
	"container/list"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"
)

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------------------------SessionDatabase implementation---------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

type (
	// Session the configuration for sessions
	// has 5 fields
	// first is the cookieName, the session's name (string) ["mysessionsecretcookieid"]
	// second enable if you want to decode the cookie's key also
	// third is the time which the client's cookie expires
	// forth is the gcDuration (time.Duration) when this time passes it removes the unused sessions from the memory until the user come back
	// fifth is the DisableSubdomainPersistence which you can set it to true in order dissallow your q subdomains to have access to the session cook
	//
	Session struct {
		// Cookie string, the session's client cookie name, for example: "qsessionid"
		Cookie string
		// DecodeCookie set it to true to decode the cookie key with base64 URLEncoding
		// Defaults to false
		DecodeCookie bool
		// Expires the duration of which the cookie must expires (created_time.Add(Expires)).
		// If you want to delete the cookie when the browser closes, set it to -1 but in this case, the server side's session duration is up to GcDuration
		//
		// Default infinitive/unlimited life duration(0)

		Expires time.Duration
		// GcDuration every how much duration(GcDuration) the memory should be clear for unused cookies (GcDuration)
		// for example: time.Duration(2)*time.Hour. it will check every 2 hours if cookie hasn't be used for 2 hours,
		// deletes it from backend memory until the user comes back, then the session continue to work as it was
		//
		// Default 2 hours
		GcDuration time.Duration

		// DisableSubdomainPersistence set it to true in order dissallow your q subdomains to have access to the session cookie
		// defaults to false
		DisableSubdomainPersistence bool
		// UseSessionDB registers a session database, you can register more than one
		// accepts a session database which implements a Load(sid string) map[string]interface{} and an Update(sid string, newValues map[string]interface{})
		// the only reason that a session database will be useful for you is when you want to keep the session's values/data after the app restart
		// a session database doesn't have write access to the session, it doesn't accept the context, so forget 'cookie database' for sessions, I will never allow that, for your protection.
		//
		// Note: Don't worry if no session database is registered, your context.Session will continue to work.
		Databases Databases
	}
	// SessionDatabase is the interface which all session databases should implement
	// By design it doesn't support any type of cookie store like other frameworks, I want to protect you, believe me, no context access (although we could)
	// The scope of the database is to store somewhere the sessions in order to keep them after restarting the server, nothing more.
	// the values are stored by the underline session, the check for new sessions, or 'this session value should added' are made automatically by q, you are able just to set the values to your backend database with Load function.
	// session database doesn't have any write or read access to the session, the loading of the initial data is done by the Load(string) map[string]interfface{} function
	// synchronization are made automatically, you can register more than one session database but the first non-empty Load return data will be used as the session values.
	SessionDatabase interface {
		Load(string) map[string]interface{}
		Update(string, map[string]interface{})
	}
	// Databases a slice of SessionDatabase
	Databases []SessionDatabase
)

func (s Session) newManager() *sessionsManager {
	if s.Cookie == "" { // means disable sessions
		return nil
	}

	if s.GcDuration <= 0 {
		s.GcDuration = time.Duration(2) * time.Hour
	}
	// init and start the sess manager
	sess := newSessionsManager(s)

	// register all available session databases
	for i := range s.Databases {
		sess.registerDatabase(s.Databases[i])
	}

	return sess
}

// SessionStore is  session's store interface
// implemented by the internal sessionStore iteral, normally the end-user will never use this interface.
type SessionStore interface {
	ID() string
	Get(string) interface{}
	GetString(key string) string
	GetInt(key string) int
	GetAll() map[string]interface{}
	VisitAll(cb func(k string, v interface{}))
	Set(string, interface{})
	Delete(string)
	Clear()
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------------------------Session implementation-----------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

// sessionStore is an 'object' which wraps the session provider with its session databases, only frontend user has access to this session object.
// this is really used on context and everywhere inside q
// implements the SessionStore interface
type sessionStore struct {
	sid              string
	values           map[string]interface{} // here is the real values
	mu               sync.Mutex
	lastAccessedTime time.Time
	createdAt        time.Time
	provider         *sessionProvider
}

// ID returns the session's id
func (s *sessionStore) ID() string {
	return s.sid
}

// Get returns the value of an entry by its key
func (s *sessionStore) Get(key string) interface{} {
	s.provider.update(s.sid)
	if value, found := s.values[key]; found {
		return value
	}
	return nil
}

// GetString same as Get but returns as string, if nil then returns an empty string
func (s *sessionStore) GetString(key string) string {
	if value := s.Get(key); value != nil {
		if v, ok := value.(string); ok {
			return v
		}

	}

	return ""
}

// GetInt same as Get but returns as int, if nil then returns -1
func (s *sessionStore) GetInt(key string) int {
	if value := s.Get(key); value != nil {
		if v, ok := value.(int); ok {
			return v
		}
	}

	return -1
}

// GetAll returns all session's values
func (s *sessionStore) GetAll() map[string]interface{} {
	return s.values
}

// VisitAll loop each one entry and calls the callback function func(key,value)
func (s *sessionStore) VisitAll(cb func(k string, v interface{})) {
	for key := range s.values {
		cb(key, s.values[key])
	}
}

// Set fills the session with an entry, it receives a key and a value
// returns an error, which is always nil
func (s *sessionStore) Set(key string, value interface{}) {
	s.mu.Lock()
	s.values[key] = value
	s.mu.Unlock()
	s.provider.update(s.sid)
}

// Delete removes an entry by its key
// returns an error, which is always nil
func (s *sessionStore) Delete(key string) {
	s.mu.Lock()
	delete(s.values, key)
	s.mu.Unlock()
	s.provider.update(s.sid)
}

// Clear removes all entries
func (s *sessionStore) Clear() {
	s.mu.Lock()
	for key := range s.values {
		delete(s.values, key)
	}
	s.mu.Unlock()
	s.provider.update(s.sid)
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------------------------sessionProvider implementation---------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

type (
	// sessionProvider contains the temp sessions memory and the databases
	sessionProvider struct {
		mu        sync.Mutex
		sessions  map[string]*list.Element // underline TEMPORARY memory store used to give advantage on sessions used more times than others
		list      *list.List               // for GC
		databases []SessionDatabase
		expires   time.Duration
	}
)

func (p *sessionProvider) registerDatabase(db SessionDatabase) {
	p.mu.Lock() // for any case
	p.databases = append(p.databases, db)
	p.mu.Unlock()
}

func (p *sessionProvider) newSession(sid string) *sessionStore {

	sess := &sessionStore{
		sid:              sid,
		provider:         p,
		lastAccessedTime: time.Now(),
		values:           p.loadSessionValues(sid),
	}
	if p.expires > 0 { // if not unlimited life duration and no -1 (cookie remove action is based on browser's session)
		time.AfterFunc(p.expires, func() {
			// the destroy makes the check if this session is exists then or not,
			// this is used to destroy the session from the server-side also
			// it's good to have here for security reasons, I didn't add it on the gc function to separate its action
			p.destroy(sid)

		})
	}

	return sess

}

func (p *sessionProvider) loadSessionValues(sid string) map[string]interface{} {

	for i, n := 0, len(p.databases); i < n; i++ {
		if dbValues := p.databases[i].Load(sid); dbValues != nil && len(dbValues) > 0 {
			return dbValues // return the first non-empty from the registered stores.
		}
	}
	values := make(map[string]interface{})
	return values
}

func (p *sessionProvider) updateDatabases(sid string, newValues map[string]interface{}) {
	for i, n := 0, len(p.databases); i < n; i++ {
		p.databases[i].Update(sid, newValues)
	}
}

// Init creates the session  and returns it
func (p *sessionProvider) init(sid string) *sessionStore {
	newSession := p.newSession(sid)
	elem := p.list.PushBack(newSession)
	p.mu.Lock()
	p.sessions[sid] = elem
	p.mu.Unlock()
	return newSession
}

// Read returns the store which sid parameter is belongs
func (p *sessionProvider) read(sid string) *sessionStore {
	p.mu.Lock()
	if elem, found := p.sessions[sid]; found {
		p.mu.Unlock() // yes defer is slow
		elem.Value.(*sessionStore).lastAccessedTime = time.Now()
		return elem.Value.(*sessionStore)
	}
	p.mu.Unlock()
	// if not found create new
	sess := p.init(sid)
	return sess
}

// Destroy destroys the session, removes all sessions values, the session itself and updates the registered session databases, this called from sessionManager which removes the client's cookie also.
func (p *sessionProvider) destroy(sid string) {
	p.mu.Lock()
	if elem, found := p.sessions[sid]; found {
		sess := elem.Value.(*sessionStore)
		sess.values = nil
		p.updateDatabases(sid, nil)
		delete(p.sessions, sid)
		p.list.Remove(elem)
	}
	p.mu.Unlock()
}

// Update updates the lastAccessedTime, and moves the memory place element to the front
// always returns a nil error, for now
func (p *sessionProvider) update(sid string) {
	p.mu.Lock()
	if elem, found := p.sessions[sid]; found {
		sess := elem.Value.(*sessionStore)
		sess.lastAccessedTime = time.Now()
		p.list.MoveToFront(elem)
		p.updateDatabases(sid, sess.values)
	}
	p.mu.Unlock()
}

// GC clears the memory
func (p *sessionProvider) gc(duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for {
		elem := p.list.Back()
		if elem == nil {
			break
		}

		// if the time has passed. session was expired, then delete the session and its memory place
		// we are not destroy the session completely for the case this is re-used after
		sess := elem.Value.(*sessionStore)
		if time.Now().After(sess.lastAccessedTime.Add(duration)) {
			p.list.Remove(elem)
		} else {
			break
		}
	}
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------------------------sessionsManager implementation---------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

type (
	// sessionsManager implements the ISessionsManager interface
	// contains the cookie's name, the provider and a duration for GC and cookie life expire
	sessionsManager struct {
		config   Session
		provider *sessionProvider
	}
)

// newSessionsManager creates & returns a new SessionsManager and start its GC
func newSessionsManager(c Session) *sessionsManager {
	if c.DecodeCookie {
		c.Cookie = base64.URLEncoding.EncodeToString([]byte(c.Cookie)) // change the cookie's name/key to a more safe(?)
		// get the real value for your tests by:
		//sessIdKey := url.QueryEscape(base64.URLEncoding.EncodeToString([]byte(Sessions.Cookie)))
	}
	manager := &sessionsManager{config: c, provider: &sessionProvider{list: list.New(), sessions: make(map[string]*list.Element, 0), databases: make([]SessionDatabase, 0), expires: c.Expires}}
	//run the GC here
	go manager.gc()
	return manager
}

func (m *sessionsManager) registerDatabase(db SessionDatabase) {
	m.provider.expires = m.config.Expires // updae the expires confiuration field for any case
	m.provider.registerDatabase(db)
}

func (m *sessionsManager) generateSessionID() string {
	return base64.URLEncoding.EncodeToString(Random(32))
}

// Start starts the session
func (m *sessionsManager) start(ctx *Context) *sessionStore {
	var session *sessionStore

	cookieValue := ctx.GetCookie(m.config.Cookie)

	if cookieValue == "" { // cookie doesn't exists, let's generate a session and add set a cookie
		sid := m.generateSessionID()
		session = m.provider.init(sid)
		cookie := &http.Cookie{}
		// The RFC makes no mention of encoding url value, so here I think to encode both sessionid key and the value using the safe(to put and to use as cookie) url-encoding
		cookie.Name = m.config.Cookie
		cookie.Value = sid
		cookie.Path = "/"
		if !m.config.DisableSubdomainPersistence {

			requestDomain := ctx.Request.Host
			if portIdx := strings.IndexByte(requestDomain, ':'); portIdx > 0 {
				requestDomain = requestDomain[0:portIdx]
			}
			if validCookieDomain(requestDomain) {

				// RFC2109, we allow level 1 subdomains, but no further
				// if we have localhost.com , we want the localhost.com.
				// so if we have something like: mysubdomain.localhost.com we want the localhost here
				// if we have mysubsubdomain.mysubdomain.localhost.com we want the .mysubdomain.localhost.com here
				// slow things here, especially the 'replace' but this is a good and understable( I hope) way to get the be able to set cookies from subdomains & domain with 1-level limit
				if dotIdx := strings.LastIndexByte(requestDomain, '.'); dotIdx > 0 {
					// is mysubdomain.localhost.com || mysubsubdomain.mysubdomain.localhost.com
					s := requestDomain[0:dotIdx] // set mysubdomain.localhost || mysubsubdomain.mysubdomain.localhost
					if secondDotIdx := strings.LastIndexByte(s, '.'); secondDotIdx > 0 {
						//is mysubdomain.localhost ||  mysubsubdomain.mysubdomain.localhost
						s = s[secondDotIdx+1:] // set to localhost || mysubdomain.localhost
					}
					// replace the s with the requestDomain before the domain's siffux
					subdomainSuff := strings.LastIndexByte(requestDomain, '.')
					if subdomainSuff > len(s) { // if it is actual exists as subdomain suffix
						requestDomain = strings.Replace(requestDomain, requestDomain[0:subdomainSuff], s, 1) // set to localhost.com || mysubdomain.localhost.com
					}
				}
				// finally set the .localhost.com (for(1-level) || .mysubdomain.localhost.com (for 2-level subdomain allow)
				cookie.Domain = "." + requestDomain // . to allow persistance
			}

		}
		cookie.HttpOnly = true
		if m.config.Expires == 0 {
			// unlimited life
			cookie.Expires = CookieExpireUnlimited
		} else if m.config.Expires > 0 {
			cookie.Expires = time.Now().Add(m.config.Expires)
		} // if it's -1 then the cookie is deleted when the browser closes

		ctx.AddCookie(cookie)
		//ReleaseCookie(cookie)
	} else {
		session = m.provider.read(cookieValue)
	}
	return session
}

// Destroy kills the session and remove the associated cookie
func (m *sessionsManager) destroy(ctx *Context) {
	cookieValue := ctx.GetCookie(m.config.Cookie)
	if cookieValue == "" { // nothing to destroy
		return
	}
	ctx.RemoveCookie(m.config.Cookie)
	m.provider.destroy(cookieValue)
}

// GC tick-tock for the store cleanup
// it's a blocking function, so run it with go routine, it's totally safe
func (m *sessionsManager) gc() {
	m.provider.gc(m.config.GcDuration)
	// set a timer for the next GC
	time.AfterFunc(m.config.GcDuration, func() {
		m.gc()
	})
}
