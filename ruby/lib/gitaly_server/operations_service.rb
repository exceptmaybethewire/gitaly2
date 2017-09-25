module GitalyServer
  class OperationsService < Gitaly::OperationService::Service
    include Utils

    def user_delete_tag(request, call)
      repo = Gitlab::Git::Repository.from_call(call)

      gitaly_user = request.user
      raise GRPC::InvalidArgument.new('empty user') unless gitaly_user
      user = Gitlab::Git::User.from_gitaly(gitaly_user)

      tag_name = request.tag_name
      raise GRPC::InvalidArgument.new('empty tag name') if tag_name.blank?

      repo.rm_tag(tag_name, user: user)

      Gitaly::UserDeleteTagResponse.new
    rescue Gitlab::Git::HooksService::PreReceiveError => e
      raise GRPC::FailedPrecondition.new(e.to_s)
    end

    def user_create_branch(request, call)
      repo = Gitlab::Git::Repository.from_call(call)
      target = request.start_point
      raise GRPC::InvalidArgument.new('empty start_point') if target.empty?
      gitaly_user = request.user
      raise GRPC::InvalidArgument.new('empty user') unless gitaly_user

      branch_name = request.branch_name
      user = Gitlab::Git::User.from_gitaly(gitaly_user)
      created_branch = repo.add_branch(branch_name, user: user, target: target)
      return Gitaly::UserCreateBranchResponse.new unless created_branch

      rugged_commit = created_branch.dereferenced_target.rugged_commit
      commit = gitaly_commit_from_rugged(rugged_commit)
      branch = Gitaly::Branch.new(name: branch_name, target_commit: commit)
      Gitaly::UserCreateBranchResponse.new(branch: branch)
    rescue Gitlab::Git::Repository::InvalidRef, Gitlab::Git::CommitError => ex
      raise GRPC::FailedPrecondition.new(ex.message)
    rescue Gitlab::Git::HooksService::PreReceiveError => ex
      return Gitaly::UserCreateBranchResponse.new(pre_receive_error: ex.message)
    end
  end
end
