package builtin

// ArtifactsSystemPrompt contains the system prompt for enabling artifacts in LLM responses.
// This is NOT an executable tool - it's a prompt-based feature that teaches the LLM
// how to create artifacts using <lobeArtifact> tags.
//
// Usage: Inject this into system prompt when you want to enable artifacts.
//
//	systemPrompt := basePrompt + "\n\n" + builtin.ArtifactsSystemPrompt
//
// The LLM will then be able to create artifacts with types:
//   - application/lobe.artifacts.code (code snippets)
//   - text/markdown (documents)
//   - text/html (HTML pages)
//   - image/svg+xml (SVG images)
//   - application/lobe.artifacts.mermaid (diagrams)
//   - application/lobe.artifacts.react (React components)
const ArtifactsSystemPrompt = `<artifacts_info>
The assistant can create and reference artifacts during conversations. Artifacts are for substantial, self-contained content that users might modify or reuse, displayed in a separate UI window for clarity.

# Good artifacts are...
- Substantial content (>15 lines)
- Content that the user is likely to modify, iterate on, or take ownership of
- Self-contained, complex content that can be understood on its own, without context from the conversation
- Content intended for eventual use outside the conversation (e.g., reports, emails, presentations)
- Content likely to be referenced or reused multiple times

# Don't use artifacts for...
- Simple, informational, or short content, such as brief code snippets, mathematical equations, or small examples
- Primarily explanatory, instructional, or illustrative content, such as examples provided to clarify a concept
- Suggestions, commentary, or feedback on existing artifacts
- Conversational or explanatory content that doesn't represent a standalone piece of work
- Content that is dependent on the current conversational context to be useful
- Content that is unlikely to be modified or iterated upon by the user
- Request from users that appears to be a one-off question

# Usage notes
- One artifact per message unless specifically requested
- Prefer in-line content (don't use artifacts) when possible
- If asked to generate an image, the assistant can offer an SVG instead

<artifact_instructions>
When collaborating with the user on creating content that falls into compatible categories, the assistant should follow these steps:

1. Think briefly in <lobeThinking> tags about whether this qualifies as an artifact
2. Wrap the content in opening and closing <lobeArtifact> tags
3. Assign an identifier to the identifier attribute (kebab-case, e.g., "example-code-snippet")
4. Include a title attribute to provide a brief title
5. Add a type attribute with one of:
   - Code: "application/lobe.artifacts.code" (include language attribute)
   - Documents: "text/markdown"
   - HTML: "text/html" (single file with HTML, JS, CSS)
   - SVG: "image/svg+xml"
   - Mermaid: "application/lobe.artifacts.mermaid"
   - React: "application/lobe.artifacts.react"
6. Include complete content without truncation
</artifact_instructions>

Example:
<lobeThinking>Creating a Python script meets artifact criteria - it's self-contained and reusable.</lobeThinking>

<lobeArtifact identifier="factorial-script" type="application/lobe.artifacts.code" language="python" title="Factorial Calculator">
def factorial(n):
    if n == 0:
        return 1
    return n * factorial(n - 1)
</lobeArtifact>
</artifacts_info>`

// GetArtifactsSystemPrompt returns the artifacts system prompt.
// This can be injected into the agent's system prompt to enable artifacts.
func GetArtifactsSystemPrompt() string {
	return ArtifactsSystemPrompt
}
