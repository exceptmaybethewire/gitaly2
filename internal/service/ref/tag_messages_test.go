package ref

import (
	"io"
	"strings"
	"testing"

	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestSuccessfulGetTagMessagesRequest(t *testing.T) {
	server, serverSocketPath := runRefServiceServer(t)
	defer server.Stop()

	client, conn := newRefServiceClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	ctx, cancel := testhelper.Context()
	defer cancel()

	message1 := strings.Repeat("a", helper.MaxCommitOrTagMessageSize*2)
	message2 := strings.Repeat("b", helper.MaxCommitOrTagMessageSize)

	tag1ID := testhelper.CreateTag(t, testRepoPath, "big-tag-1", "master", &testhelper.CreateTagOpts{Message: message1})
	tag2ID := testhelper.CreateTag(t, testRepoPath, "big-tag-2", "master~", &testhelper.CreateTagOpts{Message: message2})

	request := &pb.GetTagMessagesRequest{
		Repository: testRepo,
		TagIds:     []string{tag1ID, tag2ID},
	}

	c, err := client.GetTagMessages(ctx, request)
	require.NoError(t, err)

	expectedMessages := []*pb.GetTagMessagesResponse{
		{
			TagId:   tag1ID,
			Message: []byte(message1 + "\n"),
		},
		{
			TagId:   tag2ID,
			Message: []byte(message2 + "\n"),
		},
	}
	fetchedMessages := readAllMessagesFromClient(t, c)

	require.Equal(t, expectedMessages, fetchedMessages)
}

func TestFailedGetTagMessagesRequest(t *testing.T) {
	server, serverSocketPath := runRefServiceServer(t)
	defer server.Stop()

	client, conn := newRefServiceClient(t, serverSocketPath)
	defer conn.Close()

	testCases := []struct {
		desc    string
		request *pb.GetTagMessagesRequest
		code    codes.Code
	}{
		{
			desc: "empty Repository",
			request: &pb.GetTagMessagesRequest{
				Repository: nil,
				TagIds:     []string{"5937ac0a7beb003549fc5fd26fc247adbce4a52e"},
			},
			code: codes.InvalidArgument,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			ctx, cancel := testhelper.Context()
			defer cancel()

			c, err := client.GetTagMessages(ctx, testCase.request)
			require.NoError(t, err)

			for {
				_, err = c.Recv()
				if err != nil {
					break
				}
			}

			testhelper.RequireGrpcError(t, err, testCase.code)
		})
	}
}

func readAllMessagesFromClient(t *testing.T, c pb.RefService_GetTagMessagesClient) (messages []*pb.GetTagMessagesResponse) {
	for {
		resp, err := c.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		if resp.TagId != "" {
			messages = append(messages, resp)
			// first message contains a chunk of the message, so no need to append anything
			continue
		}

		currentMessage := messages[len(messages)-1]
		currentMessage.Message = append(currentMessage.Message, resp.Message...)
	}

	return
}
