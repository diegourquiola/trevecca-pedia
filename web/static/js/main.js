// Dark mode toggle functionality
function toggleDarkMode() {
    if (document.documentElement.classList.contains('dark')) {
        document.documentElement.classList.remove('dark')
        localStorage.theme = 'light'
    } else {
        document.documentElement.classList.add('dark')
        localStorage.theme = 'dark'
    }
}

// Search input focus effects
const searchInput = document.querySelector('input[name="q"]')
if (searchInput) {
    const searchContainer = searchInput.closest('.group')
    if (searchContainer) {
        searchInput.addEventListener('focus', () => {
            searchContainer.classList.add('scale-[1.02]')
        })
        
        searchInput.addEventListener('blur', () => {
            searchContainer.classList.remove('scale-[1.02]')
        })
    }
}

// Smooth scroll for anchor links
document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function (e) {
        e.preventDefault()
        const target = document.querySelector(this.getAttribute('href'))
        if (target) {
            target.scrollIntoView({
                behavior: 'smooth',
                block: 'start'
            })
        }
    })
})

// Initialize auth state in navigation
updateNavAuth()

// If on profile page, fetch profile data
if (document.getElementById('profile-loading')) {
    fetchProfile()
}

// Wiki editor toolbar and live preview functionality
document.addEventListener('DOMContentLoaded', function() {
    const editTextarea = document.getElementById('edit-textarea')
    const previewContent = document.getElementById('entry')
    
    // --- Mobile tab switching ---
    function switchMobileEditTab(tab) {
        const editorPanel = document.getElementById('mobile-editor-panel')
        const previewPanel = document.getElementById('mobile-preview-panel')
        const editTabBtn = document.getElementById('mobile-edit-tab')
        const previewTabBtn = document.getElementById('mobile-preview-tab')
        
        if (!editorPanel || !previewPanel) return
        
        if (tab === 'edit') {
            // Show editor, hide preview
            editorPanel.classList.remove('hidden')
            previewPanel.classList.add('hidden')
            previewPanel.classList.remove('flex')
            
            // Update tab styles
            editTabBtn.classList.add('border-neutral-900', 'dark:border-neutral-100', 'text-neutral-900', 'dark:text-neutral-100')
            editTabBtn.classList.remove('border-transparent', 'text-neutral-500', 'dark:text-neutral-400')
            previewTabBtn.classList.remove('border-neutral-900', 'dark:border-neutral-100', 'text-neutral-900', 'dark:text-neutral-100')
            previewTabBtn.classList.add('border-transparent', 'text-neutral-500', 'dark:text-neutral-400')
        } else {
            // Show preview, hide editor
            editorPanel.classList.add('hidden')
            previewPanel.classList.remove('hidden')
            previewPanel.classList.add('flex')
            
            // Update tab styles
            previewTabBtn.classList.add('border-neutral-900', 'dark:border-neutral-100', 'text-neutral-900', 'dark:text-neutral-100')
            previewTabBtn.classList.remove('border-transparent', 'text-neutral-500', 'dark:text-neutral-400')
            editTabBtn.classList.remove('border-neutral-900', 'dark:border-neutral-100', 'text-neutral-900', 'dark:text-neutral-100')
            editTabBtn.classList.add('border-transparent', 'text-neutral-500', 'dark:text-neutral-400')
        }
    }
    
    // Expose function globally for onclick handlers
    window.switchMobileEditTab = switchMobileEditTab
    
    if (editTextarea && previewContent) {
        // Debounce function to limit API calls
        let debounceTimer
        function debounce(func, wait) {
            clearTimeout(debounceTimer)
            debounceTimer = setTimeout(func, wait)
        }
        
        // Function to update preview
        async function updatePreview() {
            const content = editTextarea.value
            
            // Don't show preview for empty content
            if (!content || content.trim() === '') {
                previewContent.innerHTML = '<p class="text-gray-400 italic">Start typing to see preview...</p>'
                return
            }
            
            try {
                const response = await fetch('/update-preview', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ content: content })
                })
                
                if (response.ok) {
                    const data = await response.json()
                    previewContent.innerHTML = data.html
                } else {
                    previewContent.innerHTML = '<p class="text-red-500">Failed to generate preview</p>'
                }
            } catch (error) {
                previewContent.innerHTML = '<p class="text-red-500">Error connecting to preview service</p>'
            }
        }
        
        // Initial preview load
        updatePreview()
        
        // Update preview on input with debouncing
        editTextarea.addEventListener('input', function() {
            debounce(updatePreview, 300)
        })

        // --- Toolbar formatting helpers ---

        // Wraps selected text with a prefix/suffix (e.g. **bold**)
        // If nothing is selected, inserts placeholder text wrapped with prefix/suffix
        function wrapSelection(prefix, suffix, placeholder) {
            const start = editTextarea.selectionStart
            const end = editTextarea.selectionEnd
            const text = editTextarea.value
            const selected = text.substring(start, end)

            let replacement
            let cursorStart, cursorEnd

            if (selected.length > 0) {
                // If selected text is already wrapped, unwrap it
                const before = text.substring(Math.max(0, start - prefix.length), start)
                const after = text.substring(end, end + suffix.length)
                if (before === prefix && after === suffix) {
                    editTextarea.value = text.substring(0, start - prefix.length) + selected + text.substring(end + suffix.length)
                    cursorStart = start - prefix.length
                    cursorEnd = cursorStart + selected.length
                } else {
                    replacement = prefix + selected + suffix
                    editTextarea.value = text.substring(0, start) + replacement + text.substring(end)
                    cursorStart = start + prefix.length
                    cursorEnd = cursorStart + selected.length
                }
            } else {
                replacement = prefix + placeholder + suffix
                editTextarea.value = text.substring(0, start) + replacement + text.substring(end)
                cursorStart = start + prefix.length
                cursorEnd = cursorStart + placeholder.length
            }

            editTextarea.focus()
            editTextarea.setSelectionRange(cursorStart, cursorEnd)
            editTextarea.dispatchEvent(new Event('input'))
        }

        // Inserts a prefix at the beginning of the current line (e.g. ## for headings)
        // If the line already has the prefix, it removes it (toggle behavior)
        function prefixLine(prefix, placeholder) {
            const start = editTextarea.selectionStart
            const text = editTextarea.value

            // Find start of current line
            const lineStart = text.lastIndexOf('\n', start - 1) + 1
            const lineEnd = text.indexOf('\n', start)
            const actualLineEnd = lineEnd === -1 ? text.length : lineEnd
            const line = text.substring(lineStart, actualLineEnd)

            let newLine, cursorStart, cursorEnd

            if (line.startsWith(prefix)) {
                // Remove prefix (toggle off)
                newLine = line.substring(prefix.length)
                editTextarea.value = text.substring(0, lineStart) + newLine + text.substring(actualLineEnd)
                cursorStart = lineStart
                cursorEnd = lineStart + newLine.length
            } else {
                // Strip any existing heading prefix (# through ######) before adding new one
                const stripped = line.replace(/^#{1,6}\s/, '')
                newLine = prefix + (stripped.length > 0 ? stripped : placeholder)
                editTextarea.value = text.substring(0, lineStart) + newLine + text.substring(actualLineEnd)
                cursorStart = lineStart + prefix.length
                cursorEnd = lineStart + newLine.length
            }

            editTextarea.focus()
            editTextarea.setSelectionRange(cursorStart, cursorEnd)
            editTextarea.dispatchEvent(new Event('input'))
        }

        // Inserts a block of text at the cursor (e.g. code block, horizontal rule)
        function insertBlock(block) {
            const start = editTextarea.selectionStart
            const end = editTextarea.selectionEnd
            const text = editTextarea.value

            // Ensure the block starts on a new line
            let pre = ''
            if (start > 0 && text[start - 1] !== '\n') {
                pre = '\n'
            }
            // Ensure there's a newline after the block
            let post = ''
            if (end < text.length && text[end] !== '\n') {
                post = '\n'
            }

            const insertion = pre + block + post
            editTextarea.value = text.substring(0, start) + insertion + text.substring(end)
            const cursor = start + insertion.length
            editTextarea.focus()
            editTextarea.setSelectionRange(cursor, cursor)
            editTextarea.dispatchEvent(new Event('input'))
        }

        // Inserts a link with selected text as the label, or placeholder text
        function insertLink() {
            const start = editTextarea.selectionStart
            const end = editTextarea.selectionEnd
            const text = editTextarea.value
            const selected = text.substring(start, end)

            let insertion
            let cursorStart, cursorEnd

            if (selected.length > 0) {
                insertion = '[' + selected + '](url)'
                editTextarea.value = text.substring(0, start) + insertion + text.substring(end)
                // Select "url" so user can type the URL
                cursorStart = start + selected.length + 3
                cursorEnd = cursorStart + 3
            } else {
                insertion = '[link text](url)'
                editTextarea.value = text.substring(0, start) + insertion + text.substring(end)
                // Select "link text"
                cursorStart = start + 1
                cursorEnd = cursorStart + 9
            }

            editTextarea.focus()
            editTextarea.setSelectionRange(cursorStart, cursorEnd)
            editTextarea.dispatchEvent(new Event('input'))
        }

        // Inserts an image with selected text as alt text, or placeholder
        function insertImage() {
            const start = editTextarea.selectionStart
            const end = editTextarea.selectionEnd
            const text = editTextarea.value
            const selected = text.substring(start, end)

            let insertion
            let cursorStart, cursorEnd

            if (selected.length > 0) {
                insertion = '![' + selected + '](image-url)'
                editTextarea.value = text.substring(0, start) + insertion + text.substring(end)
                // Select "image-url" so user can type the URL
                cursorStart = start + selected.length + 4
                cursorEnd = cursorStart + 9
            } else {
                insertion = '![alt text](image-url)'
                editTextarea.value = text.substring(0, start) + insertion + text.substring(end)
                // Select "alt text"
                cursorStart = start + 2
                cursorEnd = cursorStart + 8
            }

            editTextarea.focus()
            editTextarea.setSelectionRange(cursorStart, cursorEnd)
            editTextarea.dispatchEvent(new Event('input'))
        }

        // Map toolbar button data-action attributes to functions
        const toolbarActions = {
            'bold':           () => wrapSelection('**', '**', 'bold text'),
            'italic':         () => wrapSelection('_', '_', 'italic text'),
            'heading1':       () => prefixLine('# ', 'Heading 1'),
            'heading2':       () => prefixLine('## ', 'Heading 2'),
            'heading3':       () => prefixLine('### ', 'Heading 3'),
            'link':           () => insertLink(),
            'image':          () => insertImage(),
            'unordered-list': () => prefixLine('- ', 'List item'),
            'ordered-list':   () => prefixLine('1. ', 'List item'),
            'blockquote':     () => prefixLine('> ', 'Quote'),
            'inline-code':    () => wrapSelection('`', '`', 'code'),
            'code-block':     () => insertBlock('```\ncode\n```'),
            'horizontal-rule':() => insertBlock('---'),
        }

        // Attach click handlers to all toolbar buttons
        document.querySelectorAll('[data-action]').forEach(function(btn) {
            btn.addEventListener('click', function(e) {
                e.preventDefault()
                const action = this.getAttribute('data-action')
                if (toolbarActions[action]) {
                    toolbarActions[action]()
                }
            })
        })

        // Keyboard shortcuts
        editTextarea.addEventListener('keydown', function(e) {
            const mod = e.ctrlKey || e.metaKey
            if (!mod) return

            const shortcuts = {
                'b': 'bold',
                'i': 'italic',
                'k': 'link',
            }

            const action = shortcuts[e.key.toLowerCase()]
            if (action && toolbarActions[action]) {
                e.preventDefault()
                toolbarActions[action]()
            }
        })
    }
})

