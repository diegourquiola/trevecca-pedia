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

// Pages drawer toggle functionality
function togglePagesDrawer() {
    const drawer = document.getElementById('pages-drawer')
    const chevron = document.getElementById('drawer-chevron')
    
    if (drawer.classList.contains('hidden')) {
        drawer.classList.remove('hidden')
        chevron.style.transform = 'rotate(180deg)'
    } else {
        drawer.classList.add('hidden')
        chevron.style.transform = 'rotate(0deg)'
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

console.log('TreveccaPedia - Ready to explore!')
