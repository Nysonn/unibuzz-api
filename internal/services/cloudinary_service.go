package services

import (
	"context"
	"fmt"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

func boolPtr(b bool) *bool { return &b }

type CloudinaryService struct {
	cld *cloudinary.Cloudinary
}

func NewCloudinaryService(cloudName, apiKey, apiSecret string) (*CloudinaryService, error) {
	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, fmt.Errorf("cloudinary init: %w", err)
	}
	return &CloudinaryService{cld: cld}, nil
}

// UploadVideo uploads a video from a remote URL to Cloudinary and returns the secure URL.
func (s *CloudinaryService) UploadVideo(ctx context.Context, videoID, inputURL string) (string, error) {
	resp, err := s.cld.Upload.Upload(ctx, inputURL, uploader.UploadParams{
		ResourceType: "video",
		PublicID:     "unibuzz/videos/" + videoID,
		Overwrite:    boolPtr(true),
	})
	if err != nil {
		return "", fmt.Errorf("cloudinary video upload: %w", err)
	}
	return resp.SecureURL, nil
}

// UploadThumbnail uploads a local thumbnail file to Cloudinary and returns the secure URL.
func (s *CloudinaryService) UploadThumbnail(ctx context.Context, videoID, filePath string) (string, error) {
	resp, err := s.cld.Upload.Upload(ctx, filePath, uploader.UploadParams{
		ResourceType: "image",
		PublicID:     "unibuzz/thumbnails/" + videoID,
		Overwrite:    boolPtr(true),
	})
	if err != nil {
		return "", fmt.Errorf("cloudinary thumbnail upload: %w", err)
	}
	return resp.SecureURL, nil
}
