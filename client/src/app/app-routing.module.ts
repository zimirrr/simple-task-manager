import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { LoginComponent } from './auth/login/login.component';
import { AuthGuard } from './auth/auth.guard';
import { OauthLandingComponent } from './auth/oauth-landing/oauth-landing.component';
import { ManagerComponent } from './manager/manager.component';
import { ProjectComponent } from './project/project/project.component';
import { ProjectCreationComponent } from './project/project-creation/project-creation.component';
import { AllProjectsResolver } from './project/all-projects.resolver';
import { ProjectResolver } from './project/project.resolver';

const routes: Routes = [
  {path: '', component: LoginComponent},
  {path: 'manager', component: ManagerComponent, canActivate: [AuthGuard], resolve: {projects: AllProjectsResolver}},
  {
    path: 'project/:id',
    component: ProjectComponent,
    canActivate: [AuthGuard],
    resolve: {
      project: ProjectResolver
    }
  },
  {path: 'new-project', component: ProjectCreationComponent, canActivate: [AuthGuard]},
  {path: 'oauth-landing', component: OauthLandingComponent},
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule]
})
export class AppRoutingModule {
}
