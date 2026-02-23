// Auth utilities for TreveccaPedia

function getAuthURL() {
    return document.body.getAttribute('data-auth-url') || 'http://127.0.0.1:8083'
}

function getToken() {
    return localStorage.getItem('auth_token')
}

function getUser() {
    var user = localStorage.getItem('auth_user')
    if (!user) return null
    try {
        return JSON.parse(user)
    } catch {
        clearAuth()
        return null
    }
}

function saveAuth(token, user) {
    localStorage.setItem('auth_token', token)
    localStorage.setItem('auth_user', JSON.stringify(user))
}

function clearAuth() {
    localStorage.removeItem('auth_token')
    localStorage.removeItem('auth_user')
}

async function login(email, password) {
    var resp
    try {
        resp = await fetch(getAuthURL() + '/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password })
        })
    } catch {
        throw new Error('Cannot reach auth service. Is it running?')
    }
    var data
    try {
        data = await resp.json()
    } catch {
        throw new Error('Unexpected response from auth service')
    }
    if (!resp.ok) {
        throw new Error(data.error || 'Invalid email or password')
    }
    saveAuth(data.accessToken, data.user)
    return data
}

async function register(email, password) {
    var resp
    try {
        resp = await fetch(getAuthURL() + '/auth/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password })
        })
    } catch {
        throw new Error('Cannot reach auth service. Is it running?')
    }
    var data
    try {
        data = await resp.json()
    } catch {
        throw new Error('Unexpected response from auth service')
    }
    if (!resp.ok) {
        throw new Error(data.error || 'Registration failed')
    }
    saveAuth(data.accessToken, data.user)
    return data
}

function logout() {
    clearAuth()
    window.location.href = '/'
}

async function fetchProfile() {
    const token = getToken()
    if (!token) {
        showProfileState('unauthenticated')
        return
    }
    try {
        const resp = await fetch(getAuthURL() + '/auth/me', {
            headers: { 'Authorization': 'Bearer ' + token }
        })
        if (!resp.ok) {
            clearAuth()
            showProfileState('unauthenticated')
            return
        }
        const user = await resp.json()
        populateProfile(user)
        showProfileState('content')
    } catch {
        clearAuth()
        showProfileState('unauthenticated')
    }
}

function showProfileState(state) {
    const loading = document.getElementById('profile-loading')
    const unauth = document.getElementById('profile-unauthenticated')
    const content = document.getElementById('profile-content')
    if (!loading) return

    loading.classList.add('hidden')
    unauth.classList.add('hidden')
    content.classList.add('hidden')

    if (state === 'loading') loading.classList.remove('hidden')
    else if (state === 'unauthenticated') unauth.classList.remove('hidden')
    else if (state === 'content') content.classList.remove('hidden')
}

function populateProfile(user) {
    const emailEl = document.getElementById('profile-email')
    const rolesEl = document.getElementById('profile-roles')
    const idEl = document.getElementById('profile-id')
    if (!emailEl) return

    emailEl.textContent = user.email || ''
    idEl.textContent = user.id || ''

    rolesEl.innerHTML = ''
    const roles = user.roles || []
    if (roles.length === 0) {
        rolesEl.innerHTML = '<span class="text-sm text-neutral-500 dark:text-neutral-400">No roles assigned</span>'
    } else {
        roles.forEach(function(role) {
            const badge = document.createElement('span')
            badge.className = 'px-3 py-1 text-sm bg-neutral-100 dark:bg-neutral-700 text-neutral-700 dark:text-neutral-300 rounded-full'
            badge.textContent = role
            rolesEl.appendChild(badge)
        })
    }
}

function updateNavAuth() {
    const loginLink = document.getElementById('nav-login-link')
    const userMenu = document.getElementById('nav-user-menu')
    const userEmail = document.getElementById('nav-user-email')
    if (!loginLink || !userMenu) return

    const user = getUser()
    if (user) {
        loginLink.classList.add('hidden')
        userMenu.classList.remove('hidden')
        if (userEmail) userEmail.textContent = user.email || ''
    } else {
        loginLink.classList.remove('hidden')
        userMenu.classList.add('hidden')
    }
}

function toggleUserDropdown() {
    var dropdown = document.getElementById('user-dropdown')
    if (!dropdown) return
    dropdown.classList.toggle('hidden')
}

// Close dropdown when clicking outside
document.addEventListener('click', function(e) {
    var dropdown = document.getElementById('user-dropdown')
    var menu = document.getElementById('nav-user-menu')
    if (dropdown && menu && !menu.contains(e.target)) {
        dropdown.classList.add('hidden')
    }
})

// Tab switching for auth page
function switchAuthTab(tab) {
    var loginForm = document.getElementById('login-form')
    var registerForm = document.getElementById('register-form')
    var loginTab = document.getElementById('login-tab')
    var registerTab = document.getElementById('register-tab')
    if (!loginForm || !registerForm) return

    if (tab === 'login') {
        loginForm.classList.remove('hidden')
        registerForm.classList.add('hidden')
        loginTab.classList.add('border-neutral-900', 'dark:border-neutral-100', 'text-neutral-900', 'dark:text-neutral-100')
        loginTab.classList.remove('border-transparent', 'text-neutral-400', 'dark:text-neutral-500')
        registerTab.classList.remove('border-neutral-900', 'dark:border-neutral-100', 'text-neutral-900', 'dark:text-neutral-100')
        registerTab.classList.add('border-transparent', 'text-neutral-400', 'dark:text-neutral-500')
    } else {
        loginForm.classList.add('hidden')
        registerForm.classList.remove('hidden')
        registerTab.classList.add('border-neutral-900', 'dark:border-neutral-100', 'text-neutral-900', 'dark:text-neutral-100')
        registerTab.classList.remove('border-transparent', 'text-neutral-400', 'dark:text-neutral-500')
        loginTab.classList.remove('border-neutral-900', 'dark:border-neutral-100', 'text-neutral-900', 'dark:text-neutral-100')
        loginTab.classList.add('border-transparent', 'text-neutral-400', 'dark:text-neutral-500')
    }
}

function showError(elementId, message) {
    var el = document.getElementById(elementId)
    if (!el) return
    el.textContent = message
    el.classList.remove('hidden')
}

function hideError(elementId) {
    var el = document.getElementById(elementId)
    if (!el) return
    el.classList.add('hidden')
}

async function handleLogin(e) {
    e.preventDefault()
    hideError('login-error')
    var email = document.getElementById('login-email').value
    var password = document.getElementById('login-password').value
    try {
        await login(email, password)
        window.location.href = '/'
    } catch (err) {
        showError('login-error', err.message)
    }
}

async function handleRegister(e) {
    e.preventDefault()
    hideError('register-error')
    var email = document.getElementById('register-email').value
    var password = document.getElementById('register-password').value
    var confirm = document.getElementById('register-confirm').value
    if (password !== confirm) {
        showError('register-error', 'Passwords do not match')
        return
    }
    try {
        await register(email, password)
        window.location.href = '/'
    } catch (err) {
        showError('register-error', err.message)
    }
}
