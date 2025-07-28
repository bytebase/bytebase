package base

import (
	"context"
	"sync"

	"github.com/antlr4-go/antlr/v4"
)

const maxRecursionDepth = 66

// CodeCompletionCore is the core of code completion.
// It only relies on the ANTLR runtime and does not depend on any specific language.
type CodeCompletionCore struct {
	IgnoredTokens  map[int]bool
	PreferredRules map[int]bool

	// QueryRule and SelectItemAliasRule are used to determine the select item alias
	// for the completion candidates.
	// They can be used in GROUP BY, ORDER BY and HAVING clauses.
	QueryRule           int
	ShadowQueryRule     int
	SelectItemAliasRule int
	// CTERule is used to determine the CTE alias for the completion candidates.
	// Currently, it always uses in ShadowQueryRule for MySQL.
	CTERule int

	parser            antlr.Parser
	atn               *antlr.ATN
	candidates        *CandidatesCollection
	followSetsByState *FollowSetsByState
	// shortcutMap     map[int]map[int]RuleEndStatus
	statesProcessed int
	tokenStartIndex int
	tokens          []*Token

	callStack                  *RuleList
	lastQueryRuleContext       *RuleContext
	lastShadowQueryRuleContext *RuleContext

	// Context for cancellation support
	ctx context.Context
}

type Token struct {
	Type          int
	StartPosition int
}

// NewCodeCompletionCore creates a new CodeCompletionCore.
func NewCodeCompletionCore(
	ctx context.Context,
	parser antlr.Parser,
	ignoredTokens,
	preferredRules map[int]bool,
	followSets *FollowSetsByState,
	queryRule int,
	shadowQueryRule int,
	selectItemAliasRule int,
	cteRule int,
) *CodeCompletionCore {
	return &CodeCompletionCore{
		IgnoredTokens:       ignoredTokens,
		PreferredRules:      preferredRules,
		QueryRule:           queryRule,
		ShadowQueryRule:     shadowQueryRule,
		SelectItemAliasRule: selectItemAliasRule,
		CTERule:             cteRule,
		parser:              parser,
		atn:                 parser.GetATN(),
		followSetsByState:   followSets,
		ctx:                 ctx,
	}
}

// PipelineEntry is the entry of the pipeline.
type PipelineEntry struct {
	State      antlr.ATNState
	TokenIndex int
}

// RuleContext is the context of a rule.
type RuleContext struct {
	ID int
	// SelectItemAliases is the map of token index of select item aliases.
	// Only the QueryRule has the SelectItemAliases.
	SelectItemAliases map[int]bool
	CTEList           []int
}

// RuleList is the list of rules.
// Use a bitset to check existence of a rule in the list efficiently.
type RuleList struct {
	rules  []*RuleContext
	bitSet *antlr.BitSet
}

// NewRuleList creates a new RuleList.
func NewRuleList() *RuleList {
	return &RuleList{
		rules:  []*RuleContext{},
		bitSet: antlr.NewBitSet(),
	}
}

// Copy copies the RuleList.
func (l *RuleList) Copy() *RuleList {
	rules := make([]*RuleContext, len(l.rules))
	for i, rule := range l.rules {
		var cteList []int
		cteList = append(cteList, rule.CTEList...)
		rules[i] = &RuleContext{
			ID:                rule.ID,
			SelectItemAliases: rule.SelectItemAliases,
			CTEList:           cteList,
		}
	}
	bitSet := antlr.NewBitSet()
	bitSet.Or(l.bitSet)
	return &RuleList{
		rules:  rules,
		bitSet: bitSet,
	}
}

// Append appends the rules from the given RuleList.
func (l *RuleList) Append(r *RuleList) {
	for _, rule := range r.rules {
		l.Push(rule)
	}
}

// Contains checks if the rule exists in the list.
// Use the bitset to check the existence efficiently.
func (l *RuleList) Contains(rule int) bool {
	return l.bitSet.Contains(rule)
}

// Push appends the rule to the list.
func (l *RuleList) Push(rule *RuleContext) {
	l.rules = append(l.rules, rule)
	l.bitSet.Add(rule.ID)
}

// Pop pops the last rule from the list.
// HINT: Each Push should not push the existing rule, otherwise Pop will destroy the bitSet.
func (l *RuleList) Pop() *RuleContext {
	result := l.rules[len(l.rules)-1]
	l.bitSet.Remove(result.ID)
	l.rules = l.rules[:len(l.rules)-1]
	return result
}

// CandidatesCollection is the collection of candidates.
// There are two types of candidates: tokens and rules.
// Tokens are the tokens that can follow the caret position.
// Rules are the parser rules that can be reduced at the caret position.
type CandidatesCollection struct {
	Tokens map[int][]int
	Rules  map[int][]*RuleContext
}

type FollowSetWithPath struct {
	intervals antlr.IntervalSet
	path      *RuleList
	following []int
}

type FollowSetsList []FollowSetWithPath

func (l *FollowSetsList) Append(f FollowSetWithPath) {
	*l = append(*l, f)
}

// FollowSetsHolder is the holder of follow sets.
type FollowSetsHolder struct {
	sets     FollowSetsList
	combined antlr.IntervalSet
}

// FollowSetsByState is the map of follow sets by state.
// It is used to cache the follow sets.
// The FollowSetsByState is only dependent on the grammar,
// so that we can reuse it in multiple CodeCompletionCore calls for the same grammar.
// On the other hand, in single CodeCompletionCore call, the FollowSetsByState is also useful.
//
// For thread safety and performance, we use RWMutex to protect the map.
// The map is read frequently and written rarely.
type FollowSetsByState struct {
	rw sync.RWMutex
	m  map[int]FollowSetsHolder
}

// NewFollowSetsByState creates a new FollowSetsByState.
func NewFollowSetsByState() FollowSetsByState {
	return FollowSetsByState{
		m: map[int]FollowSetsHolder{},
	}
}

// Get thread safety gets the follow sets by state.
func (f *FollowSetsByState) Get(state int) FollowSetsHolder {
	f.rw.RLock()
	defer f.rw.RUnlock()

	return f.m[state]
}

// Set thread safety sets the follow sets by state.
func (f *FollowSetsByState) Set(state int, holder FollowSetsHolder) {
	f.rw.Lock()
	defer f.rw.Unlock()

	f.m[state] = holder
}

// CollectFollowSets collects the follow sets if needed.
func (f *FollowSetsByState) CollectFollowSets(parser antlr.Parser, startState antlr.ATNState, ignoredTokens, _ map[int]bool) {
	state := startState.GetStateNumber()
	f.rw.Lock()
	defer f.rw.Unlock()

	if _, ok := f.m[state]; ok {
		return
	}

	stop := parser.GetATN().GetRuleToStopState(startState.GetRuleIndex())
	followSets := determineFollowSets(parser, startState, stop, ignoredTokens)

	combined := antlr.NewIntervalSet()
	for _, set := range followSets {
		combined.AddAll(&set.intervals)
	}

	f.m[state] = FollowSetsHolder{
		sets:     followSets,
		combined: *combined,
	}
}

// determineFollowSets collects tokens that can follow the given ATN state.
func determineFollowSets(parser antlr.Parser, start, stop antlr.ATNState, ignoredTokens map[int]bool) FollowSetsList {
	seen := make(map[antlr.ATNState]bool)
	ruleStack := NewRuleList()
	result := FollowSetsList{}
	collectFollowSets(parser, start, stop, &result, seen, ruleStack, ignoredTokens)
	return result
}

func collectFollowSets(
	parser antlr.Parser,
	s antlr.ATNState,
	stopState antlr.ATNState,
	followSets *FollowSetsList,
	seen map[antlr.ATNState]bool,
	ruleStack *RuleList,
	ignoredTokens map[int]bool,
) {
	if _, exists := seen[s]; exists {
		return
	}

	seen[s] = true

	if s == stopState || s.GetStateType() == antlr.ATNStateRuleStop {
		intervals := antlr.NewIntervalSet()
		intervals.AddOne(antlr.TokenEpsilon)
		followSets.Append(FollowSetWithPath{
			intervals: *intervals,
			path:      ruleStack.Copy(),
			following: []int{},
		})
		return
	}

	for _, transition := range s.GetTransitions() {
		if ruleTransition, ok := transition.(*antlr.RuleTransition); ok {
			if ruleStack.Contains(ruleTransition.GetTarget().GetRuleIndex()) {
				continue
			}

			ruleStack.Push(&RuleContext{ID: ruleTransition.GetTarget().GetRuleIndex()})
			collectFollowSets(parser, transition.GetTarget(), stopState, followSets, seen, ruleStack, ignoredTokens)
			ruleStack.Pop()
		} else if predicateTransition, ok := transition.(*antlr.PredicateTransition); ok {
			if checkPredicate(parser, predicateTransition) {
				collectFollowSets(parser, transition.GetTarget(), stopState, followSets, seen, ruleStack, ignoredTokens)
			}
		} else if transition.GetIsEpsilon() {
			collectFollowSets(parser, transition.GetTarget(), stopState, followSets, seen, ruleStack, ignoredTokens)
		} else if _, ok := transition.(*antlr.WildcardTransition); ok {
			intervals := antlr.NewIntervalSet()
			intervals.AddRange(antlr.TokenMinUserTokenType, parser.GetATN().GetMaxTokenType())
			followSets.Append(FollowSetWithPath{
				intervals: *intervals,
				path:      ruleStack.Copy(),
				following: []int{},
			})
		} else {
			set := transition.GetLabel()
			if set != nil && set.Length() > 0 {
				if _, ok := transition.(*antlr.NotSetTransition); ok {
					set = set.Complement(antlr.TokenMinUserTokenType, parser.GetATN().GetMaxTokenType())
				}
				followSets.Append(FollowSetWithPath{
					intervals: *set,
					path:      ruleStack.Copy(),
					following: getFollowingTokens(transition, ignoredTokens),
				})
			}
		}
	}
}

// getFollowingTokens collects the tokens that can follow the given transition and only if there is a single token.
// It will not collect the ignored tokens.
func getFollowingTokens(transition antlr.Transition, ignoredTokens map[int]bool) []int {
	result := []int{}
	pipeline := []antlr.ATNState{transition.GetTarget()}

	for len(pipeline) > 0 {
		state := pipeline[len(pipeline)-1]
		pipeline = pipeline[:len(pipeline)-1]

		for _, transition := range state.GetTransitions() {
			if _, ok := transition.(*antlr.AtomTransition); ok {
				if !transition.GetIsEpsilon() {
					list := transition.GetLabel().ToList()
					if len(list) == 1 {
						if _, exists := ignoredTokens[list[0]]; !exists {
							result = append(result, list[0])
							pipeline = append(pipeline, transition.GetTarget())
						}
					}
				} else {
					pipeline = append(pipeline, transition.GetTarget())
				}
			}
		}
	}

	return result
}

func checkPredicate(parser antlr.Parser, predicateTransition *antlr.PredicateTransition) bool {
	return predicateTransition.GetPredicate().Evaluate(parser, antlr.ParserRuleContextEmpty)
}

type RuleEndStatus map[int]bool

// CollectCandidates collects the candidates.
func (c *CodeCompletionCore) CollectCandidates(caretTokenIndex int, context antlr.ParserRuleContext) *CandidatesCollection {
	// Reset the fields.

	c.candidates = &CandidatesCollection{
		Tokens: make(map[int][]int),
		Rules:  make(map[int][]*RuleContext),
	}
	c.statesProcessed = 0

	if context == nil {
		c.tokenStartIndex = 0
	} else {
		c.tokenStartIndex = context.GetStart().GetTokenIndex()
	}

	// Initialize the c.tokens:
	//   Set to the token types of tokenStream[ruleStartIndex, caretTokenIndex].
	c.tokens = []*Token{}
	tokenStream := c.parser.GetTokenStream()
	currentOffset := tokenStream.Index()
	tokenStream.Seek(c.tokenStartIndex)
	offset := 1
	for {
		token := tokenStream.LT(offset)
		offset++
		c.tokens = append(c.tokens, &Token{
			Type:          token.GetTokenType(),
			StartPosition: token.GetStart(),
		})

		if token.GetTokenIndex() >= caretTokenIndex || token.GetTokenType() == antlr.TokenEOF {
			break
		}
	}
	// Seek back to the original index.
	tokenStream.Seek(currentOffset)

	var startRule int
	if context == nil {
		startRule = 0
	} else {
		startRule = context.GetRuleIndex()
	}

	c.callStack = NewRuleList()
	c.fetchEndStatus(c.atn.GetRuleToStartState(startRule), 0 /* tokenIndex */, "" /* indentation */)
	return c.candidates
}

func (c *CodeCompletionCore) fetchEndStatus(startState antlr.ATNState, tokenIndex int, indentation string) RuleEndStatus {
	// Check for context cancellation
	select {
	case <-c.ctx.Done():
		return RuleEndStatus{}
	default:
	}

	if len(c.callStack.rules) > maxRecursionDepth {
		return RuleEndStatus{}
	}
	result := make(RuleEndStatus)
	c.followSetsByState.CollectFollowSets(c.parser, startState, c.IgnoredTokens, c.PreferredRules)

	followSets := c.followSetsByState.Get(startState.GetStateNumber())
	ruleContext := &RuleContext{ID: startState.GetRuleIndex()}
	c.callStack.Push(ruleContext)
	if startState.GetRuleIndex() == c.QueryRule {
		oldContext := c.lastQueryRuleContext
		c.lastQueryRuleContext = ruleContext
		defer func() {
			c.lastQueryRuleContext = oldContext
		}()
	}

	if startState.GetRuleIndex() == c.ShadowQueryRule {
		oldContext := c.lastShadowQueryRuleContext
		c.lastShadowQueryRuleContext = ruleContext
		defer func() {
			c.lastShadowQueryRuleContext = oldContext
		}()
	}

	if tokenIndex >= len(c.tokens)-1 {
		if _, exists := c.PreferredRules[startState.GetRuleIndex()]; exists {
			// If the rule is preferred, we should add it to the candidates.
			c.translateToRuleIndex(c.callStack)
		} else {
			for _, set := range followSets.sets {
				fullPath := c.callStack.Copy()
				fullPath.Append(set.path)
				// translateToRuleIndex will add the rule to the candidates if it is preferred.
				if !c.translateToRuleIndex(fullPath) {
					// If the rule is not preferred, we should add the following tokens to the candidates.
					for _, symbol := range set.intervals.ToList() {
						if _, exists := c.IgnoredTokens[symbol]; !exists {
							if _, exists := c.candidates.Tokens[symbol]; !exists {
								c.candidates.Tokens[symbol] = set.following
							} else {
								equal := len(c.candidates.Tokens[symbol]) == len(set.following)
								if equal {
									for i, item := range c.candidates.Tokens[symbol] {
										if item != set.following[i] {
											equal = false
											break
										}
									}
								}
								if !equal {
									// If the token is already in the candidates, and the following tokens are different,
									// we use an empty list to indicate that.
									c.candidates.Tokens[symbol] = []int{}
								}
							}
						}
					}
				}
			}
		}

		c.callStack.Pop()
		return RuleEndStatus{}
	}

	// If the current token and Epsilon are not in the follow sets, we should stop.
	currentSymbol := c.tokens[tokenIndex]
	if !followSets.combined.Contains(antlr.TokenEpsilon) && !followSets.combined.Contains(currentSymbol.Type) {
		c.callStack.Pop()
		return RuleEndStatus{}
	}

	var statePipeline []PipelineEntry
	var currentEntry PipelineEntry

	statePipeline = append(statePipeline, PipelineEntry{
		State:      startState,
		TokenIndex: tokenIndex,
	})

	for len(statePipeline) != 0 {
		// Check for context cancellation in the main loop
		select {
		case <-c.ctx.Done():
			c.callStack.Pop()
			return RuleEndStatus{}
		default:
		}

		currentEntry = statePipeline[len(statePipeline)-1]
		statePipeline = statePipeline[:len(statePipeline)-1]
		c.statesProcessed++

		atCaret := currentEntry.TokenIndex >= len(c.tokens)-1

		switch currentEntry.State.GetStateType() {
		case antlr.ATNStateRuleStart:
			indentation += "  "
		case antlr.ATNStateRuleStop:
			result[currentEntry.TokenIndex] = true
			continue
		default:
			// Other state types don't require special handling
		}

		for _, t := range currentEntry.State.GetTransitions() {
			switch transition := t.(type) {
			case *antlr.RuleTransition:
				endStatus := c.fetchEndStatus(transition.GetTarget(), currentEntry.TokenIndex, indentation)
				for status := range endStatus {
					statePipeline = append(statePipeline, PipelineEntry{
						State:      transition.GetFollowState(),
						TokenIndex: status,
					})
				}
				switch startState.GetRuleIndex() {
				case c.SelectItemAliasRule:
					if c.lastQueryRuleContext != nil {
						if c.lastQueryRuleContext.SelectItemAliases == nil {
							c.lastQueryRuleContext.SelectItemAliases = make(map[int]bool)
						}
						c.lastQueryRuleContext.SelectItemAliases[c.tokens[currentEntry.TokenIndex].StartPosition] = true
					}
					if c.lastShadowQueryRuleContext != nil {
						if c.lastShadowQueryRuleContext.SelectItemAliases == nil {
							c.lastShadowQueryRuleContext.SelectItemAliases = make(map[int]bool)
						}
						c.lastShadowQueryRuleContext.SelectItemAliases[c.tokens[currentEntry.TokenIndex].StartPosition] = true
					}
				case c.CTERule:
					if c.lastShadowQueryRuleContext != nil {
						c.lastShadowQueryRuleContext.CTEList = append(c.lastShadowQueryRuleContext.CTEList, c.tokens[currentEntry.TokenIndex].StartPosition)
					}
				default:
					// Other rule indices don't require special handling
				}
			case *antlr.PredicateTransition:
				if checkPredicate(c.parser, transition) {
					statePipeline = append(statePipeline, PipelineEntry{
						State:      transition.GetTarget(),
						TokenIndex: currentEntry.TokenIndex,
					})
				}
			case *antlr.WildcardTransition:
				if atCaret {
					if !c.translateToRuleIndex(c.callStack) {
						interval := antlr.NewIntervalSet()
						interval.AddRange(antlr.TokenMinUserTokenType, c.parser.GetATN().GetMaxTokenType())
						for _, symbol := range interval.ToList() {
							if _, exists := c.IgnoredTokens[symbol]; !exists {
								if _, exists := c.candidates.Tokens[symbol]; !exists {
									c.candidates.Tokens[symbol] = []int{}
								}
							}
						}
					}
				} else {
					statePipeline = append(statePipeline, PipelineEntry{
						State:      transition.GetTarget(),
						TokenIndex: currentEntry.TokenIndex + 1,
					})
				}
			default:
				if transition.GetIsEpsilon() {
					if atCaret {
						c.translateToRuleIndex(c.callStack)
					}

					statePipeline = append(statePipeline, PipelineEntry{
						State:      transition.GetTarget(),
						TokenIndex: currentEntry.TokenIndex,
					})
				}

				set := transition.GetLabel()
				if set != nil && set.Length() > 0 {
					if transition.GetSerializationType() == antlr.TransitionNOTSET {
						set = set.Complement(antlr.TokenMinUserTokenType, c.parser.GetATN().GetMaxTokenType())
					}
					if atCaret {
						if !c.translateToRuleIndex(c.callStack) {
							list := set.ToList()
							addFollowing := len(list) == 1
							for _, symbol := range list {
								if _, exists := c.IgnoredTokens[symbol]; !exists {
									if addFollowing {
										c.candidates.Tokens[symbol] = getFollowingTokens(transition, c.IgnoredTokens)
									} else {
										c.candidates.Tokens[symbol] = []int{}
									}
								}
							}
						}
					} else {
						currentSymbol := c.tokens[currentEntry.TokenIndex]
						if set.Contains(currentSymbol.Type) {
							statePipeline = append(statePipeline, PipelineEntry{
								State:      transition.GetTarget(),
								TokenIndex: currentEntry.TokenIndex + 1,
							})
						}
					}
				}
			}
		}
	}

	c.callStack.Pop()
	return result
}

func (c *CodeCompletionCore) translateToRuleIndex(ruleStack *RuleList) bool {
	if len(c.PreferredRules) == 0 {
		return false
	}

	for i, rule := range ruleStack.rules {
		if _, exists := c.PreferredRules[rule.ID]; exists {
			var path []*RuleContext
			path = append(path, ruleStack.rules[:i]...)
			addNew := true

			if candidates, exists := c.candidates.Rules[rule.ID]; exists && len(candidates) == len(path) {
				equal := true
				for j, item := range candidates {
					if item.ID != path[j].ID {
						equal = false
						break
					}
				}

				if !equal {
					addNew = false
				}
			}

			if addNew {
				c.candidates.Rules[ruleStack.rules[i].ID] = path
			}
			return true
		}
	}

	return false
}
