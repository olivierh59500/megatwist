package megatwist

import "embed"

// DCKAssetAssets shares an embedded resource with the optional DCK version.
func DCKAssetAssets() embed.FS { return assets }

// DCKAssetMusicData shares an embedded resource with the optional DCK version.
func DCKAssetMusicData() []byte { return musicData }
