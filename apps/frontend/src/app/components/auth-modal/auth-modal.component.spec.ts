import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { signal } from '@angular/core';
import { of, throwError } from 'rxjs';
import { HttpErrorResponse } from '@angular/common/http';
import { AuthModalComponent } from './auth-modal.component';
import { AuthService } from '../../services/auth.service';

describe('AuthModalComponent', () => {
  let component: AuthModalComponent;
  let fixture: ComponentFixture<AuthModalComponent>;
  let isModalOpenSignal: ReturnType<typeof signal<boolean>>;
  let authServiceSpy: any;

  beforeEach(async () => {
    isModalOpenSignal = signal<boolean>(false);
    authServiceSpy = {
      isModalOpen: isModalOpenSignal,
      openModal: jasmine.createSpy('openModal').and.callFake(() => isModalOpenSignal.set(true)),
      closeModal: jasmine.createSpy('closeModal').and.callFake(() => isModalOpenSignal.set(false)),
      requestMagicLink: jasmine.createSpy('requestMagicLink').and.returnValue(
        of({ status: 'ok', message: 'check your email' })
      )
    };

    await TestBed.configureTestingModule({
      imports: [AuthModalComponent],
      providers: [
        { provide: AuthService, useValue: authServiceSpy }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(AuthModalComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should not render dialog when modal is closed', () => {
    const dialog = fixture.nativeElement.querySelector('[role="dialog"]');
    expect(dialog).toBeNull();
  });

  it('should render dialog when modal is open', () => {
    isModalOpenSignal.set(true);
    fixture.detectChanges();

    const dialog = fixture.nativeElement.querySelector('[role="dialog"]');
    expect(dialog).toBeTruthy();
    expect(dialog.textContent).toContain('Sign in to FreqShow!');
  });

  it('should call closeModal when close button is clicked', () => {
    isModalOpenSignal.set(true);
    fixture.detectChanges();

    const closeBtn = fixture.nativeElement.querySelector('button[aria-label="Close modal"]');
    closeBtn.click();

    expect(authServiceSpy.closeModal).toHaveBeenCalled();
  });

  it('should submit email and show confirmation view on success', fakeAsync(() => {
    isModalOpenSignal.set(true);
    fixture.detectChanges();

    component.email = 'musicfan@example.com';
    component.onSubmit();
    tick();
    fixture.detectChanges();

    expect(authServiceSpy.requestMagicLink).toHaveBeenCalledWith('musicfan@example.com');
    expect(component.submittedEmail()).toBe('musicfan@example.com');

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Check your email');
    expect(compiled.textContent).toContain('musicfan@example.com');
  }));

  it('should display error message on failure', fakeAsync(() => {
    authServiceSpy.requestMagicLink.and.returnValue(
      throwError(() => new HttpErrorResponse({
        status: 400,
        error: { error: 'a valid email address is required' }
      }))
    );

    isModalOpenSignal.set(true);
    fixture.detectChanges();

    component.email = 'bad-email';
    component.onSubmit();
    tick();
    fixture.detectChanges();

    expect(component.errorMessage()).toBe('a valid email address is required');
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('a valid email address is required');
  }));

  it('should handle 429 rate limit error gracefully', fakeAsync(() => {
    authServiceSpy.requestMagicLink.and.returnValue(
      throwError(() => new HttpErrorResponse({
        status: 429,
        error: { error: 'a sign-in link was already sent recently — check your email' }
      }))
    );

    isModalOpenSignal.set(true);
    fixture.detectChanges();

    component.email = 'ratelimit@example.com';
    component.onSubmit();
    tick();
    fixture.detectChanges();

    expect(component.errorMessage()).toContain('a sign-in link was already sent recently');
  }));

  it('should reset view when "Use a different email" is clicked', fakeAsync(() => {
    isModalOpenSignal.set(true);
    fixture.detectChanges();

    component.email = 'initial@example.com';
    component.onSubmit();
    tick();
    fixture.detectChanges();

    expect(component.submittedEmail()).toBe('initial@example.com');

    component.tryAnotherEmail();
    tick(60);
    fixture.detectChanges();

    expect(component.submittedEmail()).toBeNull();
    expect(component.email).toBe('');
    expect(fixture.nativeElement.textContent).toContain('Sign in to FreqShow!');
  }));

  it('should close on Escape key when open', () => {
    isModalOpenSignal.set(true);
    fixture.detectChanges();

    component.onEscape();

    expect(authServiceSpy.closeModal).toHaveBeenCalled();
  });
});
