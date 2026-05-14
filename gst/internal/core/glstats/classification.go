package glstats

import "strings"

const (
	CategoryDraw        = "draw"
	CategoryBuffer      = "buffer"
	CategoryTexture     = "texture"
	CategoryShader      = "shader_program"
	CategoryVertexInput = "vertex_input"
	CategoryFramebuffer = "framebuffer"
	CategoryState       = "state"
	CategorySyncQuery   = "sync_query_readback"
	CategoryResource    = "resource_lifecycle"
	CategoryContext     = "context_present"
	CategoryLight       = "lightweight"
	CategoryOther       = "other"
)

var categoryLabels = map[string]string{
	CategoryDraw:        "Draw",
	CategoryBuffer:      "Buffer",
	CategoryTexture:     "Texture",
	CategoryShader:      "Shader/Program",
	CategoryVertexInput: "Vertex Input",
	CategoryFramebuffer: "Framebuffer",
	CategoryState:       "State",
	CategorySyncQuery:   "Sync/Query/Readback",
	CategoryResource:    "Resource Lifecycle",
	CategoryContext:     "Context/Present",
	CategoryLight:       "Lightweight",
	CategoryOther:       "Other",
}

type APIClass struct {
	APIName  string `json:"api_name"`
	Category string `json:"category"`
	Label    string `json:"label"`
	Family   string `json:"family"`
	Key      string `json:"key,omitempty"`
	IsKey    bool   `json:"is_key"`
}

func CategoryLabel(category string) string {
	if label, ok := categoryLabels[category]; ok {
		return label
	}
	return categoryLabels[CategoryOther]
}

func Classify(apiName string) APIClass {
	name := strings.TrimSpace(apiName)
	class := APIClass{
		APIName:  name,
		Category: CategoryOther,
		Label:    CategoryLabel(CategoryOther),
		Family:   "other",
	}
	lower := strings.ToLower(name)

	switch {
	case lower == "gldrawelements":
		class = keyClass(name, CategoryDraw, "draw_elements", "glDrawElements")
	case strings.HasPrefix(name, "glDrawElementsInstanced") ||
		strings.HasPrefix(name, "glDrawElementsBaseVertex") ||
		strings.HasPrefix(name, "glDrawRangeElements") ||
		strings.HasPrefix(name, "glDrawElementsIndirect"):
		class = keyClass(name, CategoryDraw, "draw_elements_variant", "glDrawElements")
	case lower == "gldrawarrays":
		class = keyClass(name, CategoryDraw, "draw_arrays", "glDrawArrays")
	case strings.HasPrefix(name, "glDrawArraysInstanced") ||
		strings.HasPrefix(name, "glDrawArraysIndirect"):
		class = keyClass(name, CategoryDraw, "draw_arrays_variant", "glDrawArrays")
	case strings.HasPrefix(name, "glMultiDraw"):
		class = keyClass(name, CategoryDraw, "multi_draw", "multiDraw")
	case strings.HasPrefix(name, "glDispatchCompute"):
		class = keyClass(name, CategoryDraw, "compute", "compute")
	case strings.HasPrefix(name, "glDraw") || name == "glBlitFramebuffer":
		class = APIClass{APIName: name, Category: CategoryDraw, Label: CategoryLabel(CategoryDraw), Family: "draw_other"}

	case name == "glBufferData" || name == "glBufferSubData" || name == "glMapBuffer" ||
		name == "glMapBufferRange" || name == "glUnmapBuffer" || name == "glCopyBufferSubData" ||
		name == "glGetBufferSubData" || strings.HasPrefix(name, "glNamedBuffer") ||
		strings.HasPrefix(name, "glBufferStorage"):
		class = keyClass(name, CategoryBuffer, "buffer_transfer", "bufferTransfer")
	case strings.HasPrefix(name, "glBindBuffer") || strings.HasPrefix(name, "glVertexArrayVertexBuffer"):
		class = APIClass{APIName: name, Category: CategoryBuffer, Label: CategoryLabel(CategoryBuffer), Family: "buffer_binding"}

	case strings.HasPrefix(name, "glTexImage") || strings.HasPrefix(name, "glTexSubImage") ||
		strings.HasPrefix(name, "glTexStorage") || strings.HasPrefix(name, "glCompressedTex") ||
		strings.HasPrefix(name, "glCopyTex") || strings.HasPrefix(name, "glGenerateMipmap"):
		class = keyClass(name, CategoryTexture, "texture_transfer", "textureTransfer")
	case name == "glReadPixels" || name == "glGetTexImage" || name == "glGetTextureImage":
		class = keyClass(name, CategorySyncQuery, "readback", "readback")
	case strings.HasPrefix(name, "glBindTexture") || strings.HasPrefix(name, "glActiveTexture") ||
		strings.HasPrefix(name, "glBindSampler") || strings.HasPrefix(name, "glSamplerParameter") ||
		strings.HasPrefix(name, "glTexParameter"):
		class = APIClass{APIName: name, Category: CategoryTexture, Label: CategoryLabel(CategoryTexture), Family: "texture_state"}

	case name == "glCompileShader":
		class = keyClass(name, CategoryShader, "shader_compile", "shaderCompile")
	case name == "glLinkProgram":
		class = keyClass(name, CategoryShader, "program_link", "programLink")
	case name == "glUseProgram":
		class = keyClass(name, CategoryShader, "program_use", "programUse")
	case strings.HasPrefix(name, "glCreateShader") || strings.HasPrefix(name, "glShaderSource") ||
		strings.HasPrefix(name, "glAttachShader") || strings.HasPrefix(name, "glDetachShader") ||
		strings.HasPrefix(name, "glCreateProgram") || strings.HasPrefix(name, "glValidateProgram") ||
		strings.HasPrefix(name, "glProgramBinary"):
		class = APIClass{APIName: name, Category: CategoryShader, Label: CategoryLabel(CategoryShader), Family: "shader_program_lifecycle"}
	case strings.HasPrefix(name, "glUniform") || strings.HasPrefix(name, "glProgramUniform") ||
		name == "glGetUniformLocation":
		class = APIClass{APIName: name, Category: CategoryShader, Label: CategoryLabel(CategoryShader), Family: "uniform_update"}

	case strings.HasPrefix(name, "glBindVertexArray") || strings.HasPrefix(name, "glVertexAttrib") ||
		strings.HasPrefix(name, "glEnableVertexAttrib") || strings.HasPrefix(name, "glDisableVertexAttrib") ||
		strings.HasPrefix(name, "glVertexPointer") || strings.HasPrefix(name, "glColorPointer") ||
		strings.HasPrefix(name, "glNormalPointer") || strings.HasPrefix(name, "glTexCoordPointer"):
		class = APIClass{APIName: name, Category: CategoryVertexInput, Label: CategoryLabel(CategoryVertexInput), Family: "vertex_input_state"}

	case strings.Contains(name, "Framebuffer") || strings.Contains(name, "Renderbuffer") ||
		name == "glDrawBuffers" || name == "glReadBuffer" || strings.HasPrefix(name, "glClear"):
		class = APIClass{APIName: name, Category: CategoryFramebuffer, Label: CategoryLabel(CategoryFramebuffer), Family: "render_target"}

	case name == "glFinish" || name == "glFlush" || strings.Contains(name, "Sync"):
		class = keyClass(name, CategorySyncQuery, "sync", "sync")
	case strings.Contains(name, "Query") || name == "glGetError":
		class = APIClass{APIName: name, Category: CategorySyncQuery, Label: CategoryLabel(CategorySyncQuery), Family: "query_or_error"}
	case strings.HasPrefix(name, "glGet") || strings.HasPrefix(name, "glIs"):
		class = APIClass{APIName: name, Category: CategoryLight, Label: CategoryLabel(CategoryLight), Family: "state_query"}

	case strings.HasPrefix(name, "glGen") || strings.HasPrefix(name, "glDelete") ||
		strings.HasPrefix(name, "glCreate") || strings.HasPrefix(name, "glDestroy"):
		class = APIClass{APIName: name, Category: CategoryResource, Label: CategoryLabel(CategoryResource), Family: "resource_lifecycle"}

	case strings.HasPrefix(name, "glX") || strings.HasPrefix(name, "egl") ||
		strings.Contains(name, "SwapBuffers") || strings.Contains(name, "MakeCurrent"):
		class = APIClass{APIName: name, Category: CategoryContext, Label: CategoryLabel(CategoryContext), Family: "context_present"}

	case strings.HasPrefix(name, "glEnable") || strings.HasPrefix(name, "glDisable") ||
		strings.Contains(name, "Blend") || strings.Contains(name, "Depth") ||
		strings.Contains(name, "Stencil") || strings.Contains(name, "Cull") ||
		strings.Contains(name, "Viewport") || strings.Contains(name, "Scissor") ||
		strings.Contains(name, "ColorMask") || strings.Contains(name, "PolygonMode") ||
		strings.Contains(name, "FrontFace") || strings.Contains(name, "PixelStore"):
		class = APIClass{APIName: name, Category: CategoryState, Label: CategoryLabel(CategoryState), Family: "pipeline_state"}
	}

	if class.Label == "" {
		class.Label = CategoryLabel(class.Category)
	}
	return class
}

func keyClass(apiName string, category string, family string, key string) APIClass {
	return APIClass{
		APIName:  apiName,
		Category: category,
		Label:    CategoryLabel(category),
		Family:   family,
		Key:      key,
		IsKey:    true,
	}
}
