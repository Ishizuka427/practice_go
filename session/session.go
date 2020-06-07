package session

import (
	"errors"
	"time"

	"github.com/labstack/echo"
	uuid "github.com/satori/go.uuid"
)

// ID はセッションを一意に識別するIDです。
type ID string

// Store はセッションデータと整合性トークンを保持する構造体です。
type Store struct {
	Data             map[string]string
	ConsistencyToken string
}

// Manager は Sessionの操作・管理を行います。
type Manager struct {
	stopCh    chan struct{}
	commandCh chan command
	stopGCCh  chan struct{}
}

// Start は Managerの開始を行います。
func (m *Manager) Start(echo *echo.Echo) {
	e = echo
	go m.mainLoop()
	time.Sleep(100 * time.Millisecond)
	go m.gcLoop()
}

// Stop は Managerの停止を行います。
func (m *Manager) Stop() {
	m.stopGCCh <- struct{}{}
	time.Sleep(100 * time.Millisecond)
	m.stopCh <- struct{}{}
}

// Create は セッションの作成を行います。
func (m *Manager) Create() (ID, error) {
	respCh := make(chan response, 1)
	defer close(respCh)
	cmd := command{commandCreate, nil, respCh}
	m.commandCh <- cmd
	resp := <-respCh
	var res ID
	if resp.err != nil {
		e.Logger.Debugf("Session Create Error. [%s]", resp.err)
		return res, resp.err
	}
	if res, ok := resp.result[0].(ID); ok {
		return res, nil
	}
	e.Logger.Debugf("Session Create Error. [%s]", ErrorOther)
	return res, ErrorOther
}

// LoadStore は データストアの読み出しを行います。
func (m *Manager) LoadStore(sessionID ID) (Store, error) {
	respCh := make(chan response, 1)
	defer close(respCh)
	req := []interface{}{sessionID}
	cmd := command{commandLoadStore, req, respCh}
	m.commandCh <- cmd
	resp := <-respCh
	var res Store
	if resp.err != nil {
		e.Logger.Debugf("Session[%s] Load store Error. [%s]", sessionID, resp.err)
		return res, resp.err
	}
	if res, ok := resp.result[0].(Store); ok {
		return res, nil
	}
	e.Logger.Debugf("Session[%s] Load store Error. [%s]", sessionID, ErrorOther)
	return res, ErrorOther
}

// SaveStore は データストアの保存を行います。
func (m *Manager) SaveStore(sessionID ID, sessionStore Store) error {
	respCh := make(chan response, 1)
	defer close(respCh)
	req := []interface{}{sessionID, sessionStore}
	cmd := command{commandSaveStore, req, respCh}
	m.commandCh <- cmd
	resp := <-respCh
	if resp.err != nil {
		e.Logger.Debugf("Session[%s] Save store Error. [%s]", sessionID, resp.err)
		return resp.err
	}
	return nil
}

// Delete は セッションの削除を行います。
func (m *Manager) Delete(sessionID ID) error {
	respCh := make(chan response, 1)
	defer close(respCh)
	req := []interface{}{sessionID}
	cmd := command{commandDelete, req, respCh}
	m.commandCh <- cmd
	resp := <-respCh
	if resp.err != nil {
		e.Logger.Debugf("Session[%s] Delete Error. [%s]", sessionID, resp.err)
		return resp.err
	}
	return nil
}

// DeleteExpired は 期限切れセッションの削除を行います。
func (m *Manager) DeleteExpired() error {
	respCh := make(chan response, 1)
	defer close(respCh)
	cmd := command{commandDelete, nil, respCh}
	m.commandCh <- cmd
	resp := <-respCh
	if resp.err != nil {
		e.Logger.Debugf("Session DeleteExpired Error. [%s]", resp.err)
		return resp.err
	}
	return nil
}

// Managerが返す各エラーのインスタンスを生成します。
var (
	ErrorBadParameter   = errors.New("Bad Parameter")
	ErrorNotFound       = errors.New("Not Found")
	ErrorInvalidToken   = errors.New("Invalid Token")
	ErrorInvalidCommand = errors.New("Invalid Command")
	ErrorNotImplemented = errors.New("Not Implemented")
	ErrorOther          = errors.New("Other")
)

// echoのインスタンス
var e *echo.Echo

// セッション毎の情報
type session struct {
	store  Store
	expire time.Time
}

// セッションの有効期限
const sessionExpire time.Duration = (3 * time.Minute)

// コマンド種別の定義
type commandType int

const (
	commandCreate        commandType = iota // セッションの作成
	commandLoadStore                        // データストアの読み出し
	commandSaveStore                        // データストアの保存
	commandDelete                           // セッションの削除
	commandDeleteExpired                    // 期限切れのセッションを削除
)

// コマンド実行のためのパラメータ
type command struct {
	cmdType    commandType
	req        []interface{}
	responseCh chan response
}

// コマンド実行の結果
type response struct {
	result []interface{}
	err    error
}

// Manager のメインループ処理
func (m *Manager) mainLoop() {
	sessions := make(map[ID]session)
	m.stopCh = make(chan struct{}, 1)
	m.commandCh = make(chan command, 1)
	defer close(m.commandCh)
	defer close(m.stopCh)
	e.Logger.Info("session.Manager:start")
loop:
	for {
		// 受信したコマンドによって処理を振り分ける
		select {
		case cmd := <-m.commandCh:
			switch cmd.cmdType {
			// セッションの作成
			case commandCreate:
				sessionID := ID(createSessionID())
				session := session{}
				sessionStore := Store{}
				sessionData := make(map[string]string)
				sessionStore.Data = sessionData
				sessionStore.ConsistencyToken = createToken()
				session.store = sessionStore
				session.expire = time.Now().Add(sessionExpire)
				sessions[sessionID] = session
				res := []interface{}{sessionID}
				e.Logger.Debugf("Session[%s] Create. expire[%s]", sessionID, session.expire)
				cmd.responseCh <- response{res, nil}
			// データストアの読み出し
			case commandLoadStore:
				reqSessionID, ok := cmd.req[0].(ID)
				if !ok {
					cmd.responseCh <- response{nil, ErrorBadParameter}
					break
				}
				session, ok := sessions[reqSessionID]
				if !ok {
					cmd.responseCh <- response{nil, ErrorNotFound}
					break
				}
				if time.Now().After(session.expire) {
					cmd.responseCh <- response{nil, ErrorNotFound}
					break
				}
				sessionStore := Store{}
				sessionData := make(map[string]string)
				for k, v := range session.store.Data {
					sessionData[k] = v
				}
				sessionStore.Data = sessionData
				sessionStore.ConsistencyToken = session.store.ConsistencyToken
				session.expire = time.Now().Add(sessionExpire)
				sessions[reqSessionID] = session
				e.Logger.Debugf("Session[%s] Load store. store[%s] expire[%s]", reqSessionID, session.store, session.expire)
				res := []interface{}{sessionStore}
				cmd.responseCh <- response{res, nil}
			// データストアの保存
			case commandSaveStore:
				reqSessionID, ok := cmd.req[0].(ID)
				if !ok {
					cmd.responseCh <- response{nil, ErrorBadParameter}
					break
				}
				reqSessionStore, ok := cmd.req[1].(Store)
				if !ok {
					cmd.responseCh <- response{nil, ErrorBadParameter}
					break
				}
				session, ok := sessions[reqSessionID]
				if !ok {
					cmd.responseCh <- response{nil, ErrorNotFound}
					break
				}
				if time.Now().After(session.expire) {
					cmd.responseCh <- response{nil, ErrorNotFound}
					break
				}
				if session.store.ConsistencyToken != reqSessionStore.ConsistencyToken {
					cmd.responseCh <- response{nil, ErrorInvalidToken}
					break
				}
				sessionStore := Store{}
				sessionData := make(map[string]string)
				for k, v := range reqSessionStore.Data {
					sessionData[k] = v
				}
				sessionStore.Data = sessionData
				sessionStore.ConsistencyToken = createToken()
				session.store = sessionStore
				session.expire = time.Now().Add(sessionExpire)
				sessions[reqSessionID] = session
				e.Logger.Debugf("Session[%s] Save store. store[%s] expire[%s]", reqSessionID, session.store, session.expire)
				cmd.responseCh <- response{nil, nil}
			// セッションの削除
			case commandDelete:
				reqSessionID, ok := cmd.req[0].(ID)
				if !ok {
					cmd.responseCh <- response{nil, ErrorBadParameter}
					break
				}
				session, ok := sessions[reqSessionID]
				if !ok {
					cmd.responseCh <- response{nil, ErrorNotFound}
					break
				}
				if time.Now().After(session.expire) {
					cmd.responseCh <- response{nil, ErrorNotFound}
					break
				}
				delete(sessions, reqSessionID)
				e.Logger.Debugf("Session[%s] Delete.", reqSessionID)
				cmd.responseCh <- response{nil, nil}
			// 期限切れのセッションを削除
			case commandDeleteExpired:
				e.Logger.Debugf("Run Session GC. Now[%s]", time.Now())
				for k, v := range sessions {
					if time.Now().After(v.expire) {
						e.Logger.Debugf("Session[%s] expire delete. expire[%s]", k, v.expire)
						delete(sessions, k)
					}
				}
				cmd.responseCh <- response{nil, nil}
			// それ以外（エラー）
			default:
				cmd.responseCh <- response{nil, ErrorInvalidCommand}
			}
		case <-m.stopCh:
			break loop
		}
	}
	e.Logger.Info("session.Manager:stop")
}

// 期限切れセッションの定期削除処理
func (m *Manager) gcLoop() {
	m.stopGCCh = make(chan struct{}, 1)
	defer close(m.stopGCCh)
	e.Logger.Info("session.Manager GC:start")
	t := time.NewTicker(1 * time.Minute)
loop:
	for {
		select {
		case <-t.C:
			respCh := make(chan response, 1)
			defer close(respCh)
			cmd := command{commandDeleteExpired, nil, respCh}
			m.commandCh <- cmd
			<-respCh
		case <-m.stopGCCh:
			break loop
		}
	}
	t.Stop()
	e.Logger.Info("session.Manager GC:stop")
}

// 新規セッションIDの発行
func createSessionID() string {
	return uuid.NewV4().String()
}
