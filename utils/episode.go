package utils

// CompletionThreshold is the percentage at which an episode is considered complete
const CompletionThreshold = 95.0

// IsEpisodeComplete returns true if the episode playback percentage is above the completion threshold
func IsEpisodeComplete(percentageProgress float64) bool {
	return percentageProgress >= CompletionThreshold
}

// GetNextEpisode returns the next episode number based on completion status
// If the current episode is complete (>= 95%), returns the next episode
// Otherwise, returns the current episode for resuming
func GetNextEpisode(currentEpisode, totalEpisodes int, percentageProgress float64) int {
	if IsEpisodeComplete(percentageProgress) && currentEpisode < totalEpisodes {
		return currentEpisode + 1
	}
	return currentEpisode
}
