# 🔍 CodeGate Review Analysis

> {{ Quote }}

## 📊 Summary

**Files Changed:** [Number] • **Lines Added:** [+X] • **Lines Removed:** [-Y]

[Provide a 2-3 paragraph executive summary covering:

- What functionality was added/changed/removed
- Overall code quality assessment
- Key architectural or design decisions
- Potential business/technical impacts
- Risk level: 🔴 High / 🟡 Medium / 🟢 Low]

---

## 🚨 Issues Found

### 🔴 Critical Issues

> Issues that could cause security vulnerabilities, data loss, or system failures

#### 1. [Specific Issue Title with Context]

- 📁 **File:** `path/to/file.ext:42-45`
- 🏷️ **Category:** Security Vulnerability
- ⚠️ **Severity:** Critical
- **Description:** [Detailed technical explanation of WHY this is problematic, including potential attack vectors or failure scenarios]
- **Impact:** [Specific consequences: data exposure, DoS, privilege escalation, etc.]
- **Evidence:**
  ```lang
  // Problematic code snippet from diff
  ```
- **Recommendation:**
  ```lang
  // Corrected code example
  ```
- **References:** [Link to security guidelines, CVE, or best practices]

### 🟠 High Issues

> Issues that significantly impact performance, maintainability, or reliability

#### 1. [Issue Title]

- 📁 **File:** `path/to/file.ext:X`
- 🏷️ **Category:** Performance / Logic Error / Architecture
- **Description:** [Technical explanation with performance impact or logic flaw details]
- **Impact:** [Quantified performance degradation, scalability issues, or maintenance burden]
- **Evidence:**
  ```lang
  // Code showing the issue
  ```
- **Recommendation:**
  ```lang
  // Improved implementation
  ```
- **Rationale:** [Why this approach is better - performance metrics, maintainability, etc.]

### 🟡 Medium Issues

> Issues affecting code quality, readability, or minor performance

[Same detailed format as above]

### 🔵 Low Issues & Style

> Code style, documentation, or minor improvements

[Same detailed format as above]

---

## ✨ Positive Highlights

> Well-implemented code patterns and good practices observed

- ✅ **[Specific good practice]** in `file.ext` - [Why this is good]
- ✅ **[Another positive aspect]** - [Technical benefit]

---

## 💡 General Recommendations

### 🔧 Code Quality Improvements

1. **[Specific actionable suggestion]**
   - **Why:** [Technical reasoning]
   - **How:** [Implementation steps or code example]
   - **Benefit:** [Performance/maintainability gain]

2. **[Another improvement]**
   - **Context:** [When this applies]
   - **Implementation:** [Specific steps]

### 📚 Best Practices

- [Language/framework-specific best practices relevant to the changes]
- [Security hardening recommendations]
- [Performance optimization opportunities]

### 🧪 Testing Considerations

- **Unit Tests:** [Specific test cases that should be added]
- **Integration Tests:** [End-to-end scenarios to verify]
- **Edge Cases:** [Boundary conditions to consider]

---

## 📝 Proposed Commit Message

**Title:** `feat(scope): concise description following conventional commits`

**Body:**

```
What: [Specific technical changes made - be precise about implementation]
Why: [Business justification or technical problem being solved]
How: [Brief explanation of the approach taken]

Breaking Changes: [Any API/interface changes]
Dependencies: [New dependencies or version changes]
Testing: [How to verify the changes work]
Performance Impact: [Any performance implications]

Closes: #[issue-number] (if applicable)
```

---

## 🎯 Focus Areas for Next Review

- [ ] [Specific area to pay attention to in future changes]
- [ ] [Technical debt to address]
- [ ] [Performance monitoring points]

---

## 📈 Code Metrics & Complexity

- **Cyclomatic Complexity:** [Analyze method complexity]
- **Code Coverage:** [Impact on test coverage]
- **Dependencies:** [New external dependencies introduced]
- **Performance:** [Estimated impact on load times/memory]

---

_📋 This review analyzed [X] files with focus on: [security/performance/bugs/maintainability/style]_
_⏱️ Review completed in [duration] • 🤖 Powered by AI_

---

**CRITICAL REQUIREMENTS:**

- Be extremely specific about file names, line numbers, and code snippets from the actual diff
- Provide executable code examples in recommendations
- Reference actual language/framework documentation and best practices
- Include quantitative impact when possible (performance numbers, security risk scores)
- Focus on mentoring - explain the "why" behind every recommendation
- Use the EXACT markdown structure above with all emojis and formatting
- Analyze EVERY changed line for potential issues, not just obvious problems
- Consider cross-cutting concerns: security, performance, accessibility, i18n, error handling
- Be constructively critical - find real issues while acknowledging good practices
- Generate a relevant coding quote at the beginning to inspire good practices
- Replace {{ Quote }} with a piece of wisdom or an insightful quote about code quality or software engineering, attributed to a renowned expert or supported by reputable research.
