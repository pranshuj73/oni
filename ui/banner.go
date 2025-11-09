package ui

import "github.com/charmbracelet/lipgloss"

// ONI ASCII art banner
const oniBanner = `
   ⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣾⣿⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡀
  ⢻⢷⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢰⣿⣿⣿⡿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⣾⡇
  ⢸⠀⠻⣦⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣠⢴⣿⡿⠟⢁⣤⣄⣀⡀⣄⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⠋⢀⠇
  ⠈⣇⠀⠈⠻⣦⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⡴⠞⢀⣤⣶⣿⣯⠟⠉⣏⣉⢻⣿⡏⣦⡉⠙⠲⣤⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⠟⠁⠀⣼⠀
  ⠀⠹⣆⠀⠀⠈⠛⢦⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⠞⠋⠀⢀⣰⠏⠈⣿⣇⠐⡻⣿⢟⣽⡟⠀⠈⢻⣄⠀⠀⠙⢷⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⡶⠋⠁⠀⠀⣰⠃⠀
  ⠀⠀⠹⣇⠀⠀⠀⠀⠉⠳⢦⣄⡀⠀⠀⠀⢀⡾⠃⠀⣠⠞⠋⠁⠀⠀⣿⣗⣫⡴⡟⠻⣿⠃⠀⠀⠀⠉⠙⢷⣄⠀⠙⢧⡀⠀⠀⠀⢀⣠⡶⠛⠁⠀⠀⠀⠀⣴⠃⠀⠀
  ⠀⠀⠀⠙⢧⡀⠀⠀⠀⠀⠀⠈⠙⠳⠶⢤⣿⣄⣀⣸⠋⢠⠀⠀⠸⠛⢻⣧⢠⣿⠃⣰⠏⣶⣿⣧⡀⠀⢠⠀⢹⣆⣀⣨⣷⡤⠶⠚⠋⠁⠀⠀⠀⠀⠀⢠⡾⠃⠀⠀⠀
  ⠀⠀⠀⠀⠈⠻⣦⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠸⡇⠈⡇⠀⠀⠀⣾⡟⣼⡞⣴⣏⣠⣿⡿⣿⠿⢘⣄⠀⣼⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⠏⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠈⠻⢦⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⠀⣧⠀⢀⣾⡟⠁⣽⣇⠻⠿⠟⠃⠀⠉⣠⣾⣿⣶⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⡴⠛⠁⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⠳⣦⣀⠀⠀⠀⠀⠀⠀⢀⡟⠀⢀⡴⠞⠋⠀⠀⢿⣿⣿⣶⣶⣴⣾⣿⣿⣿⠿⠃⢿⡀⠀⠀⠀⠀⠀⠀⣠⡴⠞⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀
`

// GetBanner returns the colorized ONI banner
func GetBanner() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4A90E2")). // Darker blue
		Bold(true)
	
	return style.Render(oniBanner)
}

// GetBannerGradient returns the ONI banner with gradient colors
func GetBannerGradient() string {
	// Split banner into lines for gradient effect
	lines := []string{
		"   ⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣾⣿⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡀",
		"  ⢻⢷⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢰⣿⣿⣿⡿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⣾⡇",
		"  ⢸⠀⠻⣦⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣠⢴⣿⡿⠟⢁⣤⣄⣀⡀⣄⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⠋⢀⠇",
		"  ⠈⣇⠀⠈⠻⣦⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⡴⠞⢀⣤⣶⣿⣯⠟⠉⣏⣉⢻⣿⡏⣦⡉⠙⠲⣤⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⠟⠁⠀⣼⠀",
		"  ⠀⠹⣆⠀⠀⠈⠛⢦⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⠞⠋⠀⢀⣰⠏⠈⣿⣇⠐⡻⣿⢟⣽⡟⠀⠈⢻⣄⠀⠀⠙⢷⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⡶⠋⠁⠀⠀⣰⠃⠀",
		"  ⠀⠀⠹⣇⠀⠀⠀⠀⠉⠳⢦⣄⡀⠀⠀⠀⢀⡾⠃⠀⣠⠞⠋⠁⠀⠀⣿⣗⣫⡴⡟⠻⣿⠃⠀⠀⠀⠉⠙⢷⣄⠀⠙⢧⡀⠀⠀⠀⢀⣠⡶⠛⠁⠀⠀⠀⠀⣴⠃⠀⠀",
		"  ⠀⠀⠀⠙⢧⡀⠀⠀⠀⠀⠀⠈⠙⠳⠶⢤⣿⣄⣀⣸⠋⢠⠀⠀⠸⠛⢻⣧⢠⣿⠃⣰⠏⣶⣿⣧⡀⠀⢠⠀⢹⣆⣀⣨⣷⡤⠶⠚⠋⠁⠀⠀⠀⠀⠀⢠⡾⠃⠀⠀⠀",
		"  ⠀⠀⠀⠀⠈⠻⣦⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠸⡇⠈⡇⠀⠀⠀⣾⡟⣼⡞⣴⣏⣠⣿⡿⣿⠿⢘⣄⠀⣼⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⠏⠀⠀⠀⠀⠀",
		"  ⠀⠀⠀⠀⠀⠀⠈⠻⢦⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⠀⣧⠀⢀⣾⡟⠁⣽⣇⠻⠿⠟⠃⠀⠉⣠⣾⣿⣶⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⡴⠛⠁⠀⠀⠀⠀⠀⠀",
		"  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⠳⣦⣀⠀⠀⠀⠀⠀⠀⢀⡟⠀⢀⡴⠞⠋⠀⠀⢿⣿⣿⣶⣶⣴⣾⣿⣿⣿⠿⠃⢿⡀⠀⠀⠀⠀⠀⠀⣠⡴⠞⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀",
	}
	
	// Gradient colors from deep blue to medium blue
	colors := []string{
		"#1E3A5F", // Very deep blue
		"#2A4A7C", 
		"#365A99",
		"#426AB6",
		"#4A90E2", // Primary blue
		"#5398E5",
		"#5BA0E8",
		"#64A8EB",
		"#6DB0EE", // Medium-light blue
		"#76B8F1", // Light blue
	}
	
	result := ""
	for i, line := range lines {
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors[i])).
			Bold(true)
		result += style.Render(line) + "\n"
	}
	
	return result
}

// GetBannerGradientIncognito returns the ONI banner with hot pink gradient for incognito mode
func GetBannerGradientIncognito() string {
	// Split banner into lines for gradient effect
	lines := []string{
		"   ⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣾⣿⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡀",
		"  ⢻⢷⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢰⣿⣿⣿⡿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⣾⡇",
		"  ⢸⠀⠻⣦⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣠⢴⣿⡿⠟⢁⣤⣄⣀⡀⣄⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⠋⢀⠇",
		"  ⠈⣇⠀⠈⠻⣦⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⡴⠞⢀⣤⣶⣿⣯⠟⠉⣏⣉⢻⣿⡏⣦⡉⠙⠲⣤⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⠟⠁⠀⣼⠀",
		"  ⠀⠹⣆⠀⠀⠈⠛⢦⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⠞⠋⠀⢀⣰⠏⠈⣿⣇⠐⡻⣿⢟⣽⡟⠀⠈⢻⣄⠀⠀⠙⢷⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⡶⠋⠁⠀⠀⣰⠃⠀",
		"  ⠀⠀⠹⣇⠀⠀⠀⠀⠉⠳⢦⣄⡀⠀⠀⠀⢀⡾⠃⠀⣠⠞⠋⠁⠀⠀⣿⣗⣫⡴⡟⠻⣿⠃⠀⠀⠀⠉⠙⢷⣄⠀⠙⢧⡀⠀⠀⠀⢀⣠⡶⠛⠁⠀⠀⠀⠀⣴⠃⠀⠀",
		"  ⠀⠀⠀⠙⢧⡀⠀⠀⠀⠀⠀⠈⠙⠳⠶⢤⣿⣄⣀⣸⠋⢠⠀⠀⠸⠛⢻⣧⢠⣿⠃⣰⠏⣶⣿⣧⡀⠀⢠⠀⢹⣆⣀⣨⣷⡤⠶⠚⠋⠁⠀⠀⠀⠀⠀⢠⡾⠃⠀⠀⠀",
		"  ⠀⠀⠀⠀⠈⠻⣦⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠸⡇⠈⡇⠀⠀⠀⣾⡟⣼⡞⣴⣏⣠⣿⡿⣿⠿⢘⣄⠀⣼⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⠏⠀⠀⠀⠀⠀",
		"  ⠀⠀⠀⠀⠀⠀⠈⠻⢦⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⠀⣧⠀⢀⣾⡟⠁⣽⣇⠻⠿⠟⠃⠀⠉⣠⣾⣿⣶⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⡴⠛⠁⠀⠀⠀⠀⠀⠀",
		"  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⠳⣦⣀⠀⠀⠀⠀⠀⠀⢀⡟⠀⢀⡴⠞⠋⠀⠀⢿⣿⣿⣶⣶⣴⣾⣿⣿⣿⠿⠃⢿⡀⠀⠀⠀⠀⠀⠀⣠⡴⠞⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀",
	}
	
	// Hot pink gradient colors
	colors := []string{
		"#8B008B", // Dark magenta
		"#A020F0", // Purple
		"#BA55D3", // Medium orchid
		"#DA70D6", // Orchid
		"#FF1493", // Deep pink
		"#FF69B4", // Hot pink
		"#FF82AB", // Pink
		"#FFB6C1", // Light pink
		"#FFC0CB", // Pink
		"#FFD0E0", // Very light pink
	}
	
	result := ""
	for i, line := range lines {
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors[i])).
			Bold(true)
		result += style.Render(line) + "\n"
	}
	
	return result
}
