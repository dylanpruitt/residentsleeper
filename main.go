package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
)

type keyMap struct {
	TabRight           key.Binding
	TabLeft            key.Binding
	ListPrev           key.Binding
	ListNext           key.Binding
	ListAdd            key.Binding
	ListDelete         key.Binding
	EditURL            key.Binding
	SendRequest        key.Binding
	OpenQuerySelection key.Binding
	UnfocusTextInput   key.Binding
	Help               key.Binding
	Quit               key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.TabLeft, k.UnfocusTextInput},
		{k.TabRight, k.Help},
		{k.SendRequest, k.Quit},
	}
}

var keys = keyMap{
	TabRight: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "change open tab"),
	),
	TabLeft: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "change open tab"),
	),
	ListPrev: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "focus previous header"),
	),
	ListNext: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "focus next header"),
	),
	ListAdd: key.NewBinding(
		key.WithKeys("z"),
		key.WithHelp("z", "add new header"),
	),
	ListDelete: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "delete focused header"),
	),
	EditURL: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "edit url"),
	),
	SendRequest: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "send request"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "quit"),
	),
	UnfocusTextInput: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "unfocus text input"),
	),
	OpenQuerySelection: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "select query"),
	),
}

var tabOpenStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#222222")).Background(lipgloss.Color("#ccdbdc"))
var tabClosedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#dddddd")).Background(lipgloss.Color("#007ea7"))
var responseBodyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#dddddd")).Background(lipgloss.Color("#003249"))

var responseOKStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#dddddd")).Background(lipgloss.Color("#0ead69"))
var responseClientErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#222222")).Background(lipgloss.Color("#ffd23f"))
var responseServerErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#222222")).Background(lipgloss.Color("#cb0b0a"))

const querySelectionTabWidth = 30

type HTTPMethod string

const (
	POST HTTPMethod = "POST"
	GET  HTTPMethod = "GET"
)

type UIState string

const (
	UIStateWaitingForInput     UIState = "Waiting for user input"
	UIStateSelectingQuery      UIState = "Selecting query to use"
	UIStateEditingURL          UIState = "Editing URL to send request to"
	UIStateAddingQueryParam    UIState = "Adding request query parameter"
	UIStateEditingQueryParam   UIState = "Editing request query parameter"
	UIStateEditingHeader       UIState = "Editing request header"
	UIStateAddingHeader        UIState = "Adding request header"
	UIStateWaitingForResponse  UIState = "Sent HTTP request, waiting for HTTP response"
	UIStateShowingResponse     UIState = "Received HTTP response"
	UIStateShowingRequestError UIState = "Received error sending HTTP request"
	UIStateUserQuit            UIState = "Exiting program..."
)

type UITab string

const (
	TabQueryParams UITab = "Params"
	TabHeaders     UITab = "Headers"
	TabBody        UITab = "Body"
	TabResponse    UITab = "Response"
)

type ResponseData struct {
	status      string
	header      http.Header
	body        string
	timeElapsed string
	err         error
}

type HeaderData struct {
	name  string
	value string
}

type QueryParamData struct {
	name  string
	value string
}

type QueryData struct {
	name          string
	url           string
	body          []byte
	headers       []HeaderData
	queryParams   []QueryParamData
	requestMethod HTTPMethod
	responseData  *ResponseData
}

type model struct {
	queries          []QueryData
	currentQueryData *QueryData
	uiState          UIState
	tabs             []UITab
	currentTab       UITab
	viewport         viewport.Model
	textarea         textarea.Model
	help             help.Model
	keys             keyMap
	textInput        textinput.Model
	focusedHeader    int
	focusedParam     int
	focusedQuery     int
	screenWidth      int
	mainTabWidth     int
	bodyHeight       int
}

func (m model) Init() tea.Cmd {
	return tea.SetWindowTitle("residentsleeper")
}

func (m *model) removeFocusedHeader() {
	if len(m.currentQueryData.headers) == 0 {
		return
	}
	m.currentQueryData.headers = slices.Delete(m.currentQueryData.headers, m.focusedHeader, m.focusedHeader+1)
	if m.focusedHeader == len(m.currentQueryData.headers) {
		m.focusedHeader = len(m.currentQueryData.headers) - 1
	}
}

func (m *model) removeFocusedQueryParam() {
	if len(m.currentQueryData.queryParams) == 0 {
		return
	}
	m.currentQueryData.queryParams = slices.Delete(m.currentQueryData.queryParams, m.focusedParam, m.focusedParam+1)
	if m.focusedParam == len(m.currentQueryData.queryParams) {
		m.focusedParam = len(m.currentQueryData.queryParams) - 1
	}
}

func initialModel() model {
	modelHelp := help.New()
	modelHelp.ShowAll = true

	ta := textarea.New()
	ta.SetWidth(120)
	ta.SetHeight(20)
	ta.FocusedStyle.Text = responseBodyStyle
	ta.BlurredStyle.Base = responseBodyStyle
	ta.FocusedStyle.Base = responseBodyStyle
	ta.Placeholder = "Enter request body here"
	ta.SetValue("{}\n")

	ti := textinput.New()
	ti.Placeholder = "Enter header (ex. Accept:application/json;v=2)"
	ti.CharLimit = 150
	ti.Width = 60
	ti.TextStyle = tabOpenStyle
	ti.PlaceholderStyle = tabClosedStyle
	ti.Cursor.Style = tabOpenStyle

	viewport := viewport.New(120, 20)
	viewport.YPosition = 3
	viewport.Style = responseBodyStyle

	helloQuery := QueryData{
		name: "mock server hello",
		url:  "http://localhost:8090/hello",
		body: []byte{},
		headers: []HeaderData{
			{name: "Accept", value: "application/json;v=2"},
			{name: "Content-Type", value: "application/json"},
			{name: "User-Agent", value: "dylanpruitt-go-client"},
		},
		queryParams:   []QueryParamData{},
		requestMethod: GET,
		responseData:  nil,
	}

	return model{
		queries: []QueryData{
			helloQuery,
			{
				name: "mock server headers",
				url:  "http://localhost:8090/headers",
				body: []byte{},
				headers: []HeaderData{
					{name: "Accept", value: "*/*"},
					{name: "Content-Type", value: "application/json"},
					{name: "User-Agent", value: "dylanpruitt-go-client"},
				},
				queryParams:   []QueryParamData{},
				requestMethod: GET,
				responseData:  nil,
			},
			{
				name: "mock api call v1",
				url:  "http://localhost:8090/user/1",
				body: []byte{},
				headers: []HeaderData{
					{name: "Accept", value: "application/json;v=1"},
					{name: "Content-Type", value: "application/json"},
					{name: "User-Agent", value: "dylanpruitt-go-client"},
				},
				queryParams:   []QueryParamData{},
				requestMethod: GET,
				responseData:  nil,
			},
			{
				name: "mock api call v2",
				url:  "http://localhost:8090/user/1",
				body: []byte{},
				headers: []HeaderData{
					{name: "Accept", value: "application/json;v=2"},
					{name: "Content-Type", value: "application/json"},
					{name: "User-Agent", value: "dylanpruitt-go-client"},
				},
				queryParams:   []QueryParamData{},
				requestMethod: GET,
				responseData:  nil,
			},
			{
				name: "mock api call 404",
				url:  "http://localhost:8090/user/doesntexistlmao",
				body: []byte{},
				headers: []HeaderData{
					{name: "Accept", value: "application/json;v=2"},
					{name: "Content-Type", value: "application/json"},
					{name: "User-Agent", value: "dylanpruitt-go-client"},
				},
				queryParams:   []QueryParamData{},
				requestMethod: GET,
				responseData:  nil,
			},
		},
		currentQueryData: &helloQuery,
		uiState:          UIStateSelectingQuery,
		tabs:             []UITab{TabQueryParams, TabHeaders, TabBody, TabResponse},
		currentTab:       TabHeaders,
		help:             modelHelp,
		keys:             keys,
		textarea:         ta,
		viewport:         viewport,
		textInput:        ti,
		focusedHeader:    0,
		focusedQuery:     0,
	}
}

func sendRequestFromModel(m model) tea.Cmd {
	return func() tea.Msg {
		timeStart := time.Now()
		req, err := http.NewRequest(string(m.currentQueryData.requestMethod), m.currentQueryData.url, bytes.NewBuffer(m.currentQueryData.body))
		if err != nil {
			return errMsg{err: err}
		}

		for _, header := range m.currentQueryData.headers {
			req.Header.Set(header.name, header.value)
		}

		q := req.URL.Query()
		for _, param := range m.currentQueryData.queryParams {
			q.Add(param.name, param.value)
		}
		req.URL.RawQuery = q.Encode()

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return errMsg{err: err}
		}
		defer resp.Body.Close()

		responseBodyByteSlice, _ := io.ReadAll(resp.Body)
		var responseBytesBuffer bytes.Buffer
		json.Indent(&responseBytesBuffer, responseBodyByteSlice, "", "\t")
		prettyPrintedResponseJSON := responseBytesBuffer.String()

		// simulates delay from downstream server so UI changes are slow enough to watch
		ARTIFICIAL_LATENCY := time.Second
		time.Sleep(ARTIFICIAL_LATENCY)

		timeElapsedString := time.Since(timeStart).String()

		return responseMsg(&ResponseData{
			status:      resp.Status,
			header:      resp.Header,
			body:        prettyPrintedResponseJSON,
			timeElapsed: timeElapsedString,
		})
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {

	case responseMsg:
		m.currentQueryData.responseData = msg
		m.viewport.SetContent(m.currentQueryData.responseData.body)
		m.uiState = UIStateShowingResponse
		m.currentTab = TabResponse
		return m, nil

	case errMsg:
		m.currentQueryData.responseData = &ResponseData{}
		m.currentQueryData.responseData.err = msg
		m.uiState = UIStateShowingRequestError
		return m, nil

	case tea.WindowSizeMsg:
		m.screenWidth = msg.Width
		m.mainTabWidth = m.screenWidth - querySelectionTabWidth
		m.bodyHeight = msg.Height - 5
		m.viewport.Width = m.mainTabWidth
		m.viewport.Height = m.bodyHeight
		m.textarea.SetWidth(m.mainTabWidth)
		m.textarea.SetHeight(m.bodyHeight)

	case tea.KeyMsg:
		if key.Matches(msg, m.keys.SendRequest) {
            if m.uiState == UIStateSelectingQuery && m.currentTab != TabResponse {
                m.uiState = UIStateWaitingForInput
				break
            }
			if m.uiState == UIStateEditingURL {
				m.currentQueryData.url = m.textInput.Value()
				m.textInput.Blur()
				m.uiState = UIStateWaitingForInput
				break
			}
			if m.uiState == UIStateAddingHeader {
				parsedValues := strings.Split(m.textInput.Value(), ":")
				if len(parsedValues) == 2 {
					m.currentQueryData.headers = append(m.currentQueryData.headers, HeaderData{name: parsedValues[0], value: parsedValues[1]})
				}
				m.textInput.Blur()
				m.uiState = UIStateWaitingForInput
				break
			}
			if m.uiState == UIStateEditingHeader {
				parsedValues := strings.Split(m.textInput.Value(), ":")
				if len(parsedValues) == 2 {
					m.currentQueryData.headers[m.focusedHeader] = HeaderData{name: parsedValues[0], value: parsedValues[1]}
				}
				m.textInput.Blur()
				m.uiState = UIStateWaitingForInput
				break
			}
			if m.uiState == UIStateAddingQueryParam {
				parsedValues := strings.Split(m.textInput.Value(), ":")
				if len(parsedValues) == 2 {
					m.currentQueryData.queryParams = append(m.currentQueryData.queryParams, QueryParamData{name: parsedValues[0], value: parsedValues[1]})
				}
				m.textInput.Blur()
				m.uiState = UIStateWaitingForInput
				break
			}
			if m.uiState == UIStateEditingQueryParam {
				parsedValues := strings.Split(m.textInput.Value(), ":")
				if len(parsedValues) == 2 {
					m.currentQueryData.queryParams[m.focusedParam] = QueryParamData{name: parsedValues[0], value: parsedValues[1]}
				}
				m.textInput.Blur()
				m.uiState = UIStateWaitingForInput
				break
			}
			if m.currentTab == TabHeaders {
				if m.focusedHeader < 0 || m.focusedHeader >= len(m.currentQueryData.headers) {
					break
				}
				m.uiState = UIStateEditingHeader
				focusedHeader := m.currentQueryData.headers[m.focusedHeader]
				m.textInput.SetValue(fmt.Sprintf("%s:%s", focusedHeader.name, focusedHeader.value))
				m.textInput.Focus()
				break
			}
			if m.currentTab == TabQueryParams {
				if m.focusedParam < 0 || m.focusedParam >= len(m.currentQueryData.queryParams) {
					break
				}
				m.uiState = UIStateEditingQueryParam
				focusedParam := m.currentQueryData.queryParams[m.focusedParam]
				m.textInput.SetValue(fmt.Sprintf("%s:%s", focusedParam.name, focusedParam.value))
				m.textInput.Focus()
				break
			}
			if !m.textarea.Focused() {
				m.uiState = UIStateWaitingForResponse
				return m, sendRequestFromModel(m)
			}
		}
		if key.Matches(msg, m.keys.TabRight) {
			if m.uiState == UIStateEditingURL || m.uiState == UIStateAddingHeader || m.uiState == UIStateEditingHeader {
				break
			}
			if m.currentTab == TabQueryParams {
				m.currentTab = TabHeaders
				return m, nil
			}
			if m.currentTab == TabHeaders {
				m.currentTab = TabBody
				m.textarea.Focus()
				return m, nil
			}
			if m.currentTab == TabBody {
				m.currentTab = TabResponse
				m.textarea.Blur()
				return m, nil
			}
			if m.currentTab == TabResponse {
				m.currentTab = TabQueryParams
				return m, nil
			}
		}
		if key.Matches(msg, m.keys.TabLeft) {
			if m.uiState == UIStateEditingURL || m.uiState == UIStateAddingHeader || m.uiState == UIStateEditingHeader {
				break
			}
			if m.currentTab == TabQueryParams {
				m.currentTab = TabResponse
				return m, nil
			}
			if m.currentTab == TabHeaders {
				m.currentTab = TabQueryParams
				return m, nil
			}
			if m.currentTab == TabBody {
				m.currentTab = TabHeaders
				m.textarea.Blur()
				m.textInput.Focus()
				return m, nil
			}
			if m.currentTab == TabResponse {
				m.currentTab = TabBody
				m.textarea.Focus()
				return m, nil
			}
		}
		if key.Matches(msg, m.keys.ListNext) {
			if m.uiState == UIStateSelectingQuery {
				if m.focusedQuery < len(m.queries)-1 {
					m.focusedQuery += 1
					m.currentQueryData = &m.queries[m.focusedQuery]
				}
				return m, nil
			}
			if m.currentTab == TabHeaders && m.uiState != UIStateEditingHeader && m.uiState != UIStateAddingHeader {
				if m.focusedHeader < len(m.currentQueryData.headers)-1 {
					m.focusedHeader += 1
				} else {
					m.uiState = UIStateAddingHeader
					m.textInput.SetValue("")
					m.textInput.Focus()
				}
			}
			if m.currentTab == TabQueryParams && m.uiState != UIStateEditingQueryParam && m.uiState != UIStateAddingQueryParam {
				if m.focusedParam < len(m.currentQueryData.queryParams)-1 {
					m.focusedParam += 1
				} else {
					m.uiState = UIStateAddingQueryParam
					m.textInput.SetValue("")
					m.textInput.Focus()
				}
			}
		}
		if key.Matches(msg, m.keys.ListPrev) {
			if m.uiState == UIStateSelectingQuery {
				if m.focusedQuery > 0 {
					m.focusedQuery -= 1
					m.currentQueryData = &m.queries[m.focusedQuery]
				}
				return m, nil
			}
			if m.currentTab == TabHeaders && m.focusedHeader > 0 &&
				m.uiState != UIStateEditingHeader && m.uiState != UIStateAddingHeader {
				m.focusedHeader -= 1
			}
			if m.currentTab == TabQueryParams && m.focusedParam > 0 &&
				m.uiState != UIStateEditingQueryParam && m.uiState != UIStateAddingQueryParam {
				m.focusedParam -= 1
			}
		}
		if key.Matches(msg, m.keys.ListAdd) && m.currentTab == TabHeaders && m.uiState != UIStateEditingHeader && m.uiState != UIStateAddingHeader {
			m.uiState = UIStateAddingHeader
			m.textInput.SetValue("")
			m.textInput.Focus()
		}
		if key.Matches(msg, m.keys.ListAdd) && m.currentTab == TabQueryParams && m.uiState != UIStateEditingQueryParam && m.uiState != UIStateAddingQueryParam {
			m.uiState = UIStateAddingQueryParam
			m.textInput.SetValue("")
			m.textInput.Focus()
		}
		if key.Matches(msg, m.keys.ListDelete) {
			if m.currentTab == TabHeaders && m.uiState != UIStateEditingHeader && m.uiState != UIStateAddingHeader {
				m.removeFocusedHeader()
			}
			if m.currentTab == TabQueryParams && m.uiState != UIStateEditingQueryParam && m.uiState != UIStateAddingQueryParam {
				m.removeFocusedQueryParam()
			}
		}
		if key.Matches(msg, m.keys.EditURL) && m.uiState != UIStateEditingHeader && m.uiState != UIStateAddingHeader && m.uiState != UIStateEditingURL && m.uiState != UIStateEditingQueryParam && m.uiState != UIStateAddingQueryParam {
			m.uiState = UIStateEditingURL
			m.textInput.SetValue(m.currentQueryData.url)
			m.textInput.Focus()
			return m, nil
		}
		if key.Matches(msg, m.keys.UnfocusTextInput) {
			if m.uiState == UIStateSelectingQuery {
				m.uiState = UIStateWaitingForInput
				return m, nil
			}
			if m.currentTab == TabBody {
				m.textarea.Blur()
				m.currentTab = TabHeaders
				return m, nil
			}
			if m.uiState == UIStateEditingURL || m.uiState == UIStateEditingHeader || m.uiState == UIStateAddingHeader ||
				m.uiState == UIStateEditingQueryParam || m.uiState == UIStateAddingQueryParam {
				m.textInput.Blur()
				m.uiState = UIStateWaitingForInput
			}
		}
		if key.Matches(msg, m.keys.OpenQuerySelection) {
			if m.uiState == UIStateSelectingQuery {
				m.uiState = UIStateWaitingForInput
				return m, nil
			} else {
				m.uiState = UIStateSelectingQuery
				return m, nil
			}
		}
		if key.Matches(msg, m.keys.Help) {
			m.help.ShowAll = !m.help.ShowAll
		}
		if key.Matches(msg, m.keys.Quit) {
			if m.currentTab == TabBody || m.uiState == UIStateEditingURL || m.uiState == UIStateAddingHeader || m.uiState == UIStateEditingHeader ||
				m.uiState == UIStateEditingQueryParam || m.uiState == UIStateAddingQueryParam {
				break
			}
			m.uiState = UIStateUserQuit
			return m, tea.Quit
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	s := ""
	topHeader := ""
	responseString := ""
	if (m.uiState == UIStateShowingResponse || m.uiState == UIStateSelectingQuery) && m.currentQueryData.responseData != nil {
		responseString += tabClosedStyle.Render(" -> ")

		responseStyle := responseOKStyle
		if m.currentQueryData.responseData.status[:1] == "4" {
			responseStyle = responseClientErrorStyle
		}
		if m.currentQueryData.responseData.status[:1] == "5" {
			responseStyle = responseServerErrorStyle
		}
		responseString += responseStyle.Render(m.currentQueryData.responseData.status)

		responseString += tabClosedStyle.Render(fmt.Sprintf(" %s", m.currentQueryData.responseData.timeElapsed))
	}
	if m.uiState == UIStateShowingRequestError && m.currentQueryData.responseData != nil {
		responseString += tabClosedStyle.Render(" -> ")
		responseString += responseServerErrorStyle.Render("ERROR")
	}

	urlString := m.currentQueryData.url
	if m.uiState == UIStateEditingURL {
		urlString = m.textInput.View()
	}
	topHeader += tabClosedStyle.Render(fmt.Sprintf(" %s %s%s", m.currentQueryData.requestMethod, urlString, responseString))
	topHeader += "\n"
	for _, tab := range m.tabs {
		if tab == m.currentTab {
			topHeader += tabOpenStyle.Render(fmt.Sprintf(" %s ", tab))
		} else {
			topHeader += tabClosedStyle.Render(fmt.Sprintf(" %s ", tab))
		}
	}
	s += lipgloss.Place(m.mainTabWidth, 2, lipgloss.Left, lipgloss.Top, tabClosedStyle.Render(topHeader),
		lipgloss.WithWhitespaceBackground(tabClosedStyle.GetBackground()),
	)
	s += "\n"

	switch m.currentTab {
	case TabQueryParams:
		queryTabString := ""
		if len(m.currentQueryData.queryParams) == 0 {
			queryTabString += fmt.Sprintf("(no query params will be sent, press %s/%s to add one)\n", m.keys.ListAdd.Help().Key, m.keys.ListNext.Help().Key)
		}

		for i, param := range m.currentQueryData.queryParams {
			paramString := fmt.Sprintf(" %s: %s", param.name, param.value)
			if i == m.focusedParam {
				if m.uiState == UIStateEditingQueryParam {
					queryTabString += m.textInput.View() + "\n"
				} else {
                    focusedStyle := tabOpenStyle
                    if m.uiState == UIStateSelectingQuery {
                        focusedStyle = responseBodyStyle
                    }
					queryTabString += focusedStyle.Render(paramString) + "\n"
				}
			} else {
				queryTabString += paramString + "\n"
			}
		}
		if m.uiState == UIStateAddingQueryParam {
			queryTabString += m.textInput.View() + "\n"
		}

		s += lipgloss.Place(m.mainTabWidth, m.bodyHeight, lipgloss.Left, lipgloss.Top, responseBodyStyle.Render(queryTabString),
			lipgloss.WithWhitespaceBackground(responseBodyStyle.GetBackground())) + "\n"
	case TabHeaders:
		headerTabString := ""
		if len(m.currentQueryData.headers) == 0 {
			headerTabString += "(no headers will be sent)\n"
		}

		for i, header := range m.currentQueryData.headers {
			headerString := ""
			if header.name != "Authorization" {
				headerString += fmt.Sprintf(" %s: %s", header.name, header.value)
			} else {
				headerString += " Authorization: Bearer ********"
			}
			if i == m.focusedHeader {
				if m.uiState == UIStateEditingHeader {
					headerTabString += m.textInput.View() + "\n"
				} else {
                    focusedStyle := tabOpenStyle
                    if m.uiState == UIStateSelectingQuery {
                        focusedStyle = responseBodyStyle
                    }
					headerTabString += focusedStyle.Render(headerString) + "\n"
				}
			} else {
				headerTabString += headerString + "\n"
			}
		}
		if m.uiState == UIStateAddingHeader {
			headerTabString += m.textInput.View() + "\n"
		}
		s += lipgloss.Place(m.mainTabWidth, m.bodyHeight, lipgloss.Left, lipgloss.Top, responseBodyStyle.Render(headerTabString),
			lipgloss.WithWhitespaceBackground(responseBodyStyle.GetBackground())) + "\n"
	case TabBody:
		s += lipgloss.Place(m.mainTabWidth, m.bodyHeight, lipgloss.Left, lipgloss.Top, responseBodyStyle.Render(m.textarea.View()),
			lipgloss.WithWhitespaceBackground(responseBodyStyle.GetBackground())) + "\n"
	case TabResponse:
		responseTabString := ""
		switch m.uiState {
		case UIStateWaitingForInput:
			responseTabString = "response not yet sent\n"
		case UIStateWaitingForResponse:
			responseTabString = "waiting for response...\n"
		case UIStateShowingResponse:
			responseTabString = m.viewport.View()
		case UIStateSelectingQuery:
			responseTabString = m.viewport.View()
		case UIStateShowingRequestError:
			responseTabString = fmt.Sprintf("error occurred sending request: %s\n", m.currentQueryData.responseData.err)
		}
		s += lipgloss.Place(m.mainTabWidth, m.bodyHeight, lipgloss.Left, lipgloss.Top, responseBodyStyle.Render(responseTabString),
			lipgloss.WithWhitespaceBackground(responseBodyStyle.GetBackground())) + "\n"

	}
	s += lipgloss.Place(m.mainTabWidth, 1, lipgloss.Left, lipgloss.Top, tabClosedStyle.Render(" "+string(m.uiState)),
		lipgloss.WithWhitespaceBackground(tabClosedStyle.GetBackground())) + "\n"
	s += m.help.View(m.keys)

	// query selection tab elements all use 1 char less so I can add one as a border in the JoinHorizontal call (probably hacky, but the only way I could make it work quickly).
	querySelectorString := lipgloss.Place(querySelectionTabWidth-1, 1, lipgloss.Right, lipgloss.Top, tabClosedStyle.Render("\nsaved queries"),
		lipgloss.WithWhitespaceBackground(tabClosedStyle.GetBackground())) + "\n"
	for i, query := range m.queries {
		if i == m.focusedQuery {
			focusedStyle := tabOpenStyle
			if m.uiState != UIStateSelectingQuery {
				focusedStyle = responseBodyStyle
			}
			querySelectorString += lipgloss.Place(querySelectionTabWidth-1, 1, lipgloss.Right, lipgloss.Top, focusedStyle.Render(query.name),
				lipgloss.WithWhitespaceBackground(focusedStyle.GetBackground())) + "\n"
		} else {
			querySelectorString += lipgloss.Place(querySelectionTabWidth-1, 1, lipgloss.Right, lipgloss.Top, query.name) + "\n"
		}
	}
	querySelector := lipgloss.Place(querySelectionTabWidth-1, m.bodyHeight+3, lipgloss.Right, lipgloss.Top, querySelectorString)

	s = lipgloss.JoinHorizontal(lipgloss.Top, s, " ", querySelector)

	return lipgloss.Place(m.screenWidth, m.bodyHeight+5, lipgloss.Top, lipgloss.Left, s)
}

type responseMsg *ResponseData

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Printf("Uh oh, there was an error: %v\n", err)
		os.Exit(1)
	}
}
