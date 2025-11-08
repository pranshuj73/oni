package anilist

// GraphQL query for searching anime
const SearchAnimeQuery = `
query ($search: String, $page: Int, $perPage: Int, $isAdult: Boolean) {
  Page(page: $page, perPage: $perPage) {
    media(search: $search, type: ANIME, isAdult: $isAdult) {
      id
      title {
        userPreferred
        romaji
        english
        native
      }
      coverImage {
        extraLarge
        large
        medium
      }
      startDate {
        year
        month
        day
      }
      episodes
      status
      description
      averageScore
      isAdult
    }
  }
}
`

// GraphQL query for getting user's anime list
const GetAnimeListQuery = `
query ($userId: Int, $status: MediaListStatus, $type: MediaType) {
  MediaListCollection(userId: $userId, type: $type, status: $status) {
    lists {
      entries {
        id
        mediaId
        status
        score
        progress
        media {
          id
          title {
            userPreferred
            romaji
            english
            native
          }
          coverImage {
            extraLarge
            large
            medium
          }
          startDate {
            year
            month
            day
          }
          episodes
          status
          description
          averageScore
          isAdult
        }
      }
    }
  }
}
`

// GraphQL query for getting user ID
const GetUserIDQuery = `
query {
  Viewer {
    id
  }
}
`

// GraphQL mutation for updating progress
const UpdateProgressMutation = `
mutation ($mediaId: Int, $progress: Int, $status: MediaListStatus) {
  SaveMediaListEntry(mediaId: $mediaId, progress: $progress, status: $status) {
    id
    mediaId
    status
    score
    progress
    media {
      id
      title {
        userPreferred
      }
      episodes
    }
  }
}
`

// GraphQL mutation for updating score
const UpdateScoreMutation = `
mutation ($mediaId: Int, $score: Float) {
  SaveMediaListEntry(mediaId: $mediaId, score: $score) {
    id
    mediaId
    status
    score
    progress
    media {
      id
      title {
        userPreferred
      }
    }
  }
}
`

// GraphQL mutation for updating status
const UpdateStatusMutation = `
mutation ($mediaId: Int, $status: MediaListStatus) {
  SaveMediaListEntry(mediaId: $mediaId, status: $status) {
    id
    mediaId
    status
    score
    progress
    media {
      id
      title {
        userPreferred
      }
    }
  }
}
`

// GraphQL query for getting anime info
const GetAnimeInfoQuery = `
query ($id: Int) {
  Media(id: $id, type: ANIME) {
    id
    title {
      userPreferred
      romaji
      english
      native
    }
    coverImage {
      extraLarge
      large
      medium
    }
    startDate {
      year
      month
      day
    }
    episodes
    status
    description
    averageScore
    isAdult
  }
}
`

