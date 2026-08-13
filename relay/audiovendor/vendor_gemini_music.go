package audiovendor

func init() {
	register(Descriptor{
		Name:  "gemini-music",
		Match: IsGeminiMusicOriginModel,
	})
}
