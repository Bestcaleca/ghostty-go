package font

// FaceSet holds a primary terminal font and optional fallback fonts.
type FaceSet struct {
	faces []*Face
}

// NewFaceSet creates a font stack. The first face provides terminal metrics.
func NewFaceSet(primary *Face, fallbacks ...*Face) *FaceSet {
	faces := make([]*Face, 0, 1+len(fallbacks))
	if primary != nil {
		faces = append(faces, primary)
	}
	for _, face := range fallbacks {
		if face != nil {
			faces = append(faces, face)
		}
	}
	if len(faces) == 0 {
		faces = append(faces, DefaultFace())
	}
	return &FaceSet{faces: faces}
}

// Metrics returns the primary font metrics used for terminal cell sizing.
func (s *FaceSet) Metrics() Metrics {
	return s.faces[0].Metrics()
}

// FaceForRune returns the first face that contains r.
func (s *FaceSet) FaceForRune(r rune) *Face {
	for _, face := range s.faces {
		if face.HasGlyph(r) {
			return face
		}
	}
	return s.faces[0]
}

// RasterizeGlyph renders r with the first face that contains it.
func (s *FaceSet) RasterizeGlyph(r rune) *GlyphBitmap {
	return s.FaceForRune(r).RasterizeGlyph(r)
}

// Len returns the number of loaded faces in the stack.
func (s *FaceSet) Len() int {
	return len(s.faces)
}
