// Copyright 2026, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"net/url"
	"regexp"
)

// The Video settings singletons (video.twilio.com v1 /CompositionSettings/Default and
// /RecordingSettings/Default) share one shape: where Twilio stores compositions or recordings.

const (
	videoCompositionSettingsPath = "/v1/CompositionSettings/Default"
	videoRecordingSettingsPath   = "/v1/RecordingSettings/Default"
)

// videoCredentialSidRE matches the SID of a stored credential or a public key.
var videoCredentialSidRE = regexp.MustCompile(`^CR[0-9a-fA-F]{32}$`)

// videoCheckSettings validates the optional inputs the two settings share.
func videoCheckSettings(c *checker, credentials, s3url, key *string) {
	if v := credentials; v != nil {
		c.sid("awsCredentialsSid", *v, videoCredentialSidRE, "stored credential (CR followed by 32 hex characters)")
	}
	if v := key; v != nil {
		c.sid("encryptionKeySid", *v, videoCredentialSidRE, "public key (CR followed by 32 hex characters)")
	}
	if v := s3url; v != nil && !c.unk("awsS3Url") {
		if u, err := url.Parse(*v); err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			c.fail("awsS3Url", "awsS3Url must be an absolute http or https URL of the S3 bucket and path")
		}
	}
}
