package tui

// Field — одно поле формы: Label для отображения, Key для ключа результата.
type Field struct {
	Label string
	Key   string
}

// InputKind — абстрактная клавиша (без curses-кодов; модель BRIEF-008 маппит
// curses-коды сюда). Home также покрывает Ctrl-A(1), End — Ctrl-E(5), Backspace
// — KEY_BACKSPACE/127/8: маппинг на эти константы делает BRIEF-008.
type InputKind int

const (
	Char InputKind = iota // печатная руна (ASCII 32..126), поле Rune задано
	Left
	Right
	Backspace // удалить руну ПЕРЕД caret
	Delete    // удалить руну НА caret (KEY_DC)
	Home
	End
	Tab
	ShiftTab
	Enter
	Esc
)

// Input — клавиша формы. Rune значим только при Kind==Char.
type Input struct {
	Kind InputKind
	Rune rune
}

// KeyCode возвращает не-символьную клавишу.
func KeyCode(k InputKind) Input {
	return Input{Kind: k}
}

// KeyChar возвращает символьную клавишу (Kind==Char).
func KeyChar(r rune) Input {
	return Input{Kind: Char, Rune: r}
}

// FormState — чистая модель формы. Все поля экспортированы для тестов.
type FormState struct {
	Title   string
	Fields  []Field
	Values  []string // параллельно Fields
	Cursor  int      // индекс активного поля
	Pos     []int    // позиция caret (в рунах) на поле
	Scroll  []int    // горизонтальный скролл на поле
	Message string
}

// NewFormState строит форму; initial — карта key→value презаполнения
// (пустая строка для отсутствующего ключа, как в эталоне).
func NewFormState(title string, fields []Field, initial map[string]string) *FormState {
	st := &FormState{
		Title:  title,
		Fields: fields,
		Values: make([]string, len(fields)),
		Cursor: 0,
		Pos:    make([]int, len(fields)),
		Scroll: make([]int, len(fields)),
	}
	for i, f := range fields {
		v := ""
		if initial != nil {
			v = initial[f.Key]
		}
		st.Values[i] = v
		st.Pos[i] = len([]rune(v))
	}
	return st
}

// HandleKey применяет key и возвращает (result, done). done=true означает, что
// форма завершилась: result не-nil — отправка (Enter), result==nil — отмена
// (Esc). Пока не завершилась — (nil, false).
//
// Порт render_form (дословно по семантике):
//
//	Esc        → отмена: (nil,true)
//	Tab        → Cursor=(Cursor+1)%len; Message=""
//	ShiftTab   → Cursor=(Cursor-1+len)%len; Message=""
//	Enter      → собрать map: для каждого поля, если Value=="" И Key=="keyname"
//	             — пропустить; иначе map[Key]=Value. (result, true)
//	Left       → Pos[cur]=max(0,Pos[cur]-1); Message=""
//	Right      → Pos[cur]=min(lenRunes(Value),Pos[cur]+1); Message=""
//	Home       → Pos[cur]=0; Message=""
//	End        → Pos[cur]=lenRunes(Value); Message=""
//	Backspace  → если Pos[cur]>0: удалить руну перед caret, Pos[cur]--; Message=""
//	Delete     → если Pos[cur]<lenRunes(Value): удалить руну на caret; Message=""
//	Char (32..126) → вставить руну на caret, Pos[cur]++; Message=""
//	прочее Char (вне 32..126) → игнор (кириллица/UTF-8 не вставляется, паритет)
func (st *FormState) HandleKey(key Input) (result map[string]string, done bool) {
	if len(st.Fields) == 0 {
		switch key.Kind {
		case Esc:
			return nil, true
		case Enter:
			return map[string]string{}, true
		default:
			return nil, false
		}
	}
	cur := st.Cursor
	switch key.Kind {
	case Esc:
		return nil, true
	case Tab:
		st.Cursor = (cur + 1) % len(st.Fields)
		st.Message = ""
		return nil, false
	case ShiftTab:
		st.Cursor = (cur - 1 + len(st.Fields)) % len(st.Fields)
		st.Message = ""
		return nil, false
	case Enter:
		result = make(map[string]string)
		for i, f := range st.Fields {
			val := st.Values[i]
			if val == "" && f.Key == "keyname" {
				continue
			}
			result[f.Key] = val
		}
		return result, true
	case Left:
		if st.Pos[cur] > 0 {
			st.Pos[cur]--
		}
		st.Message = ""
		return nil, false
	case Right:
		n := len([]rune(st.Values[cur]))
		if st.Pos[cur] < n {
			st.Pos[cur]++
		}
		st.Message = ""
		return nil, false
	case Home:
		st.Pos[cur] = 0
		st.Message = ""
		return nil, false
	case End:
		st.Pos[cur] = len([]rune(st.Values[cur]))
		st.Message = ""
		return nil, false
	case Backspace:
		if st.Pos[cur] > 0 {
			v := []rune(st.Values[cur])
			v = append(v[:st.Pos[cur]-1], v[st.Pos[cur]:]...)
			st.Values[cur] = string(v)
			st.Pos[cur]--
		}
		st.Message = ""
		return nil, false
	case Delete:
		if st.Pos[cur] < len([]rune(st.Values[cur])) {
			v := []rune(st.Values[cur])
			v = append(v[:st.Pos[cur]], v[st.Pos[cur]+1:]...)
			st.Values[cur] = string(v)
		}
		st.Message = ""
		return nil, false
	case Char:
		r := key.Rune
		if r >= 32 && r <= 126 {
			v := []rune(st.Values[cur])
			v = append(v[:st.Pos[cur]], append([]rune{r}, v[st.Pos[cur]:]...)...)
			st.Values[cur] = string(v)
			st.Pos[cur]++
		}
		st.Message = ""
		return nil, false
	default:
		return nil, false
	}
}

// View рендерит форму в строки (height×width). Геометрия по эталону render_form:
//
//	dh = min(len(fields)*3 + 6, height-2)
//	dw = min(110, width-2)
//	field_w = max(8, dw-8)
//
// рамка DrawBox(width-?, ...) — используй width/height полностью как задано;
// для каждого i: y=2+i*3 — Label на x=3; y+1 — значение "  "+seg на x=3;
// seg вычисляется через скролл: p=Pos[i], sc=Scroll[i]; если p<sc → sc=p;
// иначе если p>=sc+field_w → sc=p-field_w+1; обновить Scroll[i]; seg=Values[i][sc:sc+field_w].
// hint-строка на (dh-2, x=2): "Tab:next  Enter:save  Esc:cancel";
// Message (если непустой) на (dh-3, x=2).
// Активное поле (Cursor) лишь помечается в FormState, не в тексте (выделение —
// презентация BRIEF-008).
func (st *FormState) View(width, height int) []string {
	dh := min(len(st.Fields)*3+6, height-2)
	dw := min(110, width-2)
	fieldW := max(8, dw-8)
	dm := dims(dh, dw)
	grid := boxGrid(dm.w, dm.h, st.Title)
	for i := range st.Fields {
		y := 2 + i*3
		safePut(grid, y, 3, st.Fields[i].Label)
		p := st.Pos[i]
		sc := st.Scroll[i]
		if p < sc {
			sc = p
		} else if p >= sc+fieldW {
			sc = p - fieldW + 1
		}
		st.Scroll[i] = sc
		val := []rune(st.Values[i])
		end := sc + fieldW
		if end > len(val) {
			end = len(val)
		}
		seg := ""
		if sc < len(val) {
			seg = string(val[sc:end])
		}
		safePut(grid, y+1, 3, "  "+seg)
	}
	safePut(grid, dh-2, 2, "Tab:next  Enter:save  Esc:cancel")
	if st.Message != "" {
		safePut(grid, dh-3, 2, st.Message)
	}
	return gridToStrings(grid)
}
