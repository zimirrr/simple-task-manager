import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { ProjectService } from '../project.service';
import { TaskService } from '../../task/task.service';
import { Project } from '../project.material';
import { CurrentUserService } from '../../user/current-user.service';
import { UserService } from '../../user/user.service';
import { NotificationService } from '../../common/notification.service';
import { Unsubscriber } from '../../common/unsubscriber';
import { User } from '../../user/user.material';

@Component({
  selector: 'app-project',
  templateUrl: './project.component.html',
  styleUrls: ['./project.component.scss']
})
export class ProjectComponent extends Unsubscriber implements OnInit {
  public project: Project;

  constructor(
    private router: Router,
    private route: ActivatedRoute,
    private projectService: ProjectService,
    private userService: UserService,
    private taskService: TaskService,
    private currentUserService: CurrentUserService,
    private notificationService: NotificationService
  ) {
    super();
  }

  ngOnInit(): void {
    this.project = this.route.snapshot.data.project;
    this.taskService.selectTask(this.project.tasks[0]);

    this.unsubscribeLater(
      this.projectService.projectChanged.subscribe(p => {
        this.project = p;
      }),
      this.projectService.projectDeleted.subscribe(removedProjectId => {
        if (this.project.id !== removedProjectId) {
          return;
        }

        if (!this.isOwner()) {
          this.notificationService.addWarning('This project has been removed');
        }

        this.router.navigate(['/manager']);
      }),
      this.projectService.projectUserRemoved.subscribe(projectId => {
        if (this.project.id !== projectId) {
          return;
        }

        this.notificationService.addWarning('You have been removed from this project');
        this.router.navigate(['/manager']);
      })
    );
  }

  public onUserRemoved(userIdToRemove: string) {
    this.projectService.removeUser(this.project.id, userIdToRemove)
      .subscribe(() => {
      }, err => {
        console.error(err);
        this.notificationService.addError('Could not remove user');
      });
  }

  public onUserInvited(user: User) {
    this.projectService.inviteUser(this.project.id, user.uid)
      .subscribe(() => {
      }, err => {
        console.error(err);
        this.notificationService.addError('Could not invite user \'' + user.name + '\'');
      });
  }

  public isOwner(): boolean {
    return this.currentUserService.getUserId() === this.project.owner.uid;
  }
}
